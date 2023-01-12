#!/usr/bin/env bash

set -euo pipefail

handler()
{
    killall ecs-cli
}

trap handler SIGINT

export ENV=${ENV:-"staging"}

TIMEOUT=${TIMEOUT:-30}
FORCE_TIMEOUT=${FORCE_TIMEOUT:-1801}
export AWS_DEFAULT_REGION=${AWS_DEFAULT_REGION:-"us-east-1"}

SCRIPTPATH=$(cd -P -- "$(dirname -- "$0")" && pwd -P)

if [ "$ENV" != "production" ]; then
  TG_EXTERNAL=$(aws elbv2 describe-target-groups --names macaroon-${ENV}-external-tg | jq -r '.TargetGroups[].TargetGroupArn' | head -n 1)
else
  TG_EXTERNAL=""
fi

TG_INTERNAL=$(aws elbv2 describe-target-groups --names macaroon-${ENV}-internal-tg | jq -r '.TargetGroups[].TargetGroupArn' | head -n 1)
VPC=$(aws elbv2 describe-target-groups --names macaroon-${ENV}-internal-tg | jq -r '.TargetGroups[0].VpcId' | head -n 1)
echo "VPC ID: ${VPC}"

SUBNETS_TEMP=$(aws ec2 describe-subnets --filter "Name=vpc-id,Values=$VPC" | jq -r '.Subnets[] | (.CidrBlock+"|"+.SubnetId)' | sort -V | head -n 2 | cut -d"|" -f 2 | tr "\n" ",")
export SUBNETS="${SUBNETS_TEMP::-1}"
echo "Subnets: ${SUBNETS}"

export SG=$(aws ec2 describe-security-groups --filters "Name=group-name,Values=tf-infra-worker-ephemeral" | jq -r '.SecurityGroups[]|.GroupId + "|" + .VpcId' | grep ${VPC} | head -n 1 | cut -d"|" -f 1)
echo "Security groups: $SG"

export PRIVATE_DNS=${PRIVATE_DNS:-"${ENV}.boltobserver.local"}

# shellcheck disable=SC2086
ecs-cli configure -c ${ENV}-privileged -r ${AWS_DEFAULT_REGION} --default-launch-type EC2
aws ecs list-services --cluster ${ENV}-privileged || true
aws ecs list-tasks --cluster ${ENV}-privileged || true

# TAG is currently production or staging (but eventually this will be replaced unlike ENV)
export ENV
export TAG=${ENV}

if [ -z ${SENTRY_DSN+x} ]; then
  echo "SENTRY_DSN variable is unset"
  exit 1
fi

# Deploy all compose-* files
for i in $(echo $SCRIPTPATH/../ecs/compose*.y*ml | sed -r 's/[^ ]+compose-//g' | sed 's/\.ya*ml//g'); do
  echo "Deploying service $i..."

  if [ "$TG_EXTERNAL" == "" ]; then
    # shellcheck disable=SC2086
    (timeout ${FORCE_TIMEOUT} ecs-cli compose --project-name ${i} -f $SCRIPTPATH/../ecs/compose-${i}.yml -ecs-params $SCRIPTPATH/../ecs/ecs-params-${i}.yml service up \
     --private-dns-namespace ${PRIVATE_DNS} --vpc ${VPC} --enable-service-discovery --dns-type A \
     --target-groups "targetGroupArn=${TG_INTERNAL},containerPort=1339,containerName=macaroon_vault" \
     --timeout ${TIMEOUT} || { echo "ecs-cli timed-out"; exit 1; })&
  else
    # shellcheck disable=SC2086
    (timeout ${FORCE_TIMEOUT} ecs-cli compose --project-name ${i} -f $SCRIPTPATH/../ecs/compose-${i}.yml -ecs-params $SCRIPTPATH/../ecs/ecs-params-${i}.yml service up \
     --private-dns-namespace ${PRIVATE_DNS} --vpc ${VPC} --enable-service-discovery --dns-type A \
     --target-groups "targetGroupArn=${TG_INTERNAL},containerPort=1339,containerName=macaroon_vault" --target-groups "targetGroupArn=${TG_EXTERNAL},containerPort=1339,containerName=macaroon_vault" \
     --timeout ${TIMEOUT} || { echo "ecs-cli timed-out"; exit 1; })&
  fi
done

for job in $(jobs -p); do
  echo "Wating for job $job to finish..."
  wait $job
done
