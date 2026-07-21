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
          sha256 = "9fd445e7a191317dcfc99d012ab632f2cc01f12af14a44dfbaba82e0f9680365";
        };
        aarch64-linux = {
          arch = "linux_arm64";
          sha256 = "322755d11f0da11169cdb234af74ada5599046c698dccc125859505f85da2a20";
        };
        x86_64-darwin = {
          arch = "darwin_amd64";
          sha256 = "d896d2776af8b06cd4acd695ad75913040ce31234f5948688fd3c3fde53b1f75";
        };
        aarch64-darwin = {
          arch = "darwin_arm64";
          sha256 = "c88ceb34f343a2bb86960e32925c5ec43b41922ee9ede1019c5cf7d7b4097718";
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
            version = "1.2.6";
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
