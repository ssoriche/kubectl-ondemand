self:
{
  config,
  lib,
  pkgs,
  ...
}:
let
  cfg = config.programs.kubectl-ondemand;
in
{
  options.programs.kubectl-ondemand = {
    enable = lib.mkEnableOption ''
      kubectl-ondemand, a kubectl/krew plugin that analyzes why Karpenter nodes are on-demand. Installing
      it puts the `kubectl-ondemand` binary on PATH, so `kubectl ondemand` becomes available system-wide'';

    package = lib.mkOption {
      type = lib.types.package;
      default = self.packages.${pkgs.stdenv.hostPlatform.system}.kubectl-ondemand;
      defaultText = lib.literalExpression "kubectl-ondemand.packages.\${system}.kubectl-ondemand";
      description = "The kubectl-ondemand package to install.";
    };
  };

  config = lib.mkIf cfg.enable {
    environment.systemPackages = [ cfg.package ];
  };
}
