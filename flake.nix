{
  description = "nitterbot - Twitter link replacer bot";

  inputs.nixpkgs.url = "github:NixOS/nixpkgs/nixos-25.11";

  outputs = { self, nixpkgs }:
    let
      pkgs = nixpkgs.legacyPackages.x86_64-linux;
    in
    {
      packages.x86_64-linux.default = pkgs.buildGoModule {
        name = "nitterbot";
        src = ./.;

        vendorHash = "sha256-uKkAXXLyel9Cvjqu7MEpmYQpHCekBVXuz3tXSUWa3us=";

        buildInputs = [ pkgs.sqlite ];

        preBuild = ''
          export CGO_ENABLED=1
          export CGO_CFLAGS="-I${pkgs.sqlite.dev}/include"
          export CGO_LDFLAGS="-L${pkgs.sqlite}/lib"
        '';
      };

      apps.x86_64-linux.default = {
        type = "app";
        program = "${self.packages.x86_64-linux.default}/bin/nitterbot";
      };

      devShells.x86_64-linux.default = pkgs.mkShell {
        buildInputs = with pkgs; [ go sqlite gcc ];
        CGO_ENABLED = "1";
        shellHook = ''
          export CGO_CFLAGS="-I${pkgs.sqlite.dev}/include"
          export CGO_LDFLAGS="-L${pkgs.sqlite}/lib"
        '';
      };
    };
}
