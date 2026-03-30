{ pkgs ? import <nixpkgs> {}, version ? "dev" }:

pkgs.buildGoModule rec {
  pname = "tdl";
  inherit version;

  src = ./.;

  subPackages = [ "tdl" ];

  vendorHash = "sha256-Q3MwzslcVv9h3QZAfqnAYmkdVtWeJnhXqYvhZmb3hps=";

  ldflags = [
    "-s" "-w"
    "-X main.version=${version}"
  ];
}
