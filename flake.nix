{
  description = "ThreeDotsLabs CLI";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = nixpkgs.legacyPackages.${system};
        version = if self ? rev then self.rev else "dev";
      in
      {
        packages.default = pkgs.buildGoModule {
          pname = "tdl";
          version = version;
          src = ./.;

          subPackages = [ "tdl" ];

          vendorHash = "sha256-Q3MwzslcVv9h3QZAfqnAYmkdVtWeJnhXqYvhZmb3hps=";

          ldflags = [
            "-s" "-w"
            "-X main.version=${version}"
            "-X main.commit=${if self ? rev then self.rev else "dirty"}"
          ];
        };
      }
    );
}
