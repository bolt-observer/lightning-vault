{
  description = "bolt.observer lightning-vault";

  outputs = { self, nixpkgs }:
    let
      supportedSystems = [ "x86_64-linux" "x86_64-darwin" "aarch64-linux" "aarch64-darwin" ];
      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      nixpkgsFor = forAllSystems (system: import nixpkgs { inherit system; });
    in
    {
      packages = forAllSystems
        (system:
          let
            version = "v0.0.18";
            pkgs = nixpkgsFor.${system};
            ldflags = ''-ldflags "-X main.GitRevision=${version} -extldflags '-static'"'';
          in
          {
            lightning-vault = pkgs.buildGoModule
              {
                name = "lightning-vault";
                inherit version;
                src = ./.;
                vendorHash = "sha256-TSd5mhBOepoPt7YKxYg3HpeO2ujqghsCV6S25FevZ7c=";

                meta = with pkgs.lib; {
                  description = "bolt.observer lightning-vault";
                };
              };
          });

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgsFor.${system};
        in
        {
          default = pkgs.mkShell {
            buildInputs = with pkgs; [ go gopls gotools go-tools ];
          };
        });

      defaultPackage = forAllSystems (system: self.packages.${system}.lightning-vault);
    };
}
