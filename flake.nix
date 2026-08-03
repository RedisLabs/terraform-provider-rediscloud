{
  description = "Flake for Redis Cloud Terraform Provider";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixpkgs-unstable";

  outputs =
    { self, nixpkgs }:
    let
      # We only consume the Terraform CLI, so there's no need to build it from
      # source: we fetch the official prebuilt release binaries directly (see the
      # `terraform` derivation below). This also sidesteps the fact that nixpkgs
      # marks the `terraform` package as unfree (HashiCorp's BUSL license), which
      # would otherwise require enabling allowUnfree.
      # These are the per-platform archive checksums for that download.
      terraformPlatforms = {
        x86_64-linux = {
          arch = "linux_amd64";
          sha256 = "c0ed7bc32ee52ae255af9982c8c88a7a4c610485cf1d55feeb037eab75fa082c";
        };
        aarch64-linux = {
          arch = "linux_arm64";
          sha256 = "f4b4ad7c6b6088960a667e34495cae490fb072947a9ff266bf5929f5333565e4";
        };
        x86_64-darwin = {
          arch = "darwin_amd64";
          sha256 = "b310ec0e626e9799000cfc8e30247cd827cf7f8030c8e0400257c7f111e93537";
        };
        aarch64-darwin = {
          arch = "darwin_arm64";
          sha256 = "db7c33eb1a446b73a443e2c55b532845f7b70cd56100bec4c96f15cfab5f50cb";
        };
      };

      forAllSystems = nixpkgs.lib.genAttrs (builtins.attrNames terraformPlatforms);
    in
    {
      devShells = forAllSystems (
        system:
        let
          pkgs = import nixpkgs { inherit system; };

          terraform = pkgs.stdenv.mkDerivation rec {
            pname = "terraform";
            version = "1.5.7";
            src = pkgs.fetchurl {
              url = "https://releases.hashicorp.com/terraform/${version}/terraform_${version}_${
                terraformPlatforms.${system}.arch
              }.zip";
              sha256 = terraformPlatforms.${system}.sha256;
            };
            nativeBuildInputs = [ pkgs.unzip ];
            sourceRoot = ".";
            installPhase = ''
              mkdir -p $out/bin
              install -m 0755 terraform $out/bin/terraform
            '';
          };

          # Built from source (not fetched as a prebuilt release binary) so the
          # binary is compiled against the same Go toolchain as the dev shell.
          # tfproviderlintx's extended checks rely on go/analysis facts, which
          # break with "failed prerequisites" when the tool and the analysed
          # packages are built with mismatched Go versions.
          tfproviderlint = pkgs.buildGoModule rec {
            pname = "tfproviderlint";
            version = "0.31.0";
            src = pkgs.fetchFromGitHub {
              owner = "bflad";
              repo = "tfproviderlint";
              rev = "v${version}";
              hash = "sha256-Wd4DAQfvrpkOcO+rVc8IDLComhO3eKPNLltMjmj3FzE=";
            };
            vendorHash = "sha256-UCyGBn+97fbd6FvcmUA+WzrI3Vqb/b8oO1c1MB91CTA=";
            subPackages = [
              "cmd/tfproviderlint"
              "cmd/tfproviderlintx"
            ];
          };
        in
        {
          default = pkgs.mkShell {
            buildInputs = [
              terraform
              tfproviderlint

              pkgs.gnumake
              pkgs.markdownlint-cli2
              pkgs.mdq

              # GNU userland so shell tooling (e.g. scripts/release-notes.sh)
              # behaves identically on macOS (BSD by default) and Linux CI.
              pkgs.coreutils
              pkgs.gawk
              pkgs.gnugrep
              pkgs.gnused

              pkgs.age
              pkgs.jq

              pkgs.go_1_25
              pkgs.golangci-lint
              pkgs.gotools
              pkgs.govulncheck
              pkgs.goreleaser
              pkgs.terraform-plugin-docs
            ];
          };
        }
      );
    };
}
