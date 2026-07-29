{
  lib,
  buildGoModule,
  go_1_26,
  versionCheckHook,
}:

let
  version = "0.1.0";
in
# Pin the Go toolchain to 1.26 to satisfy the `go 1.26` directive in go.mod;
# a sandboxed Nix build cannot download a newer toolchain on demand.
(buildGoModule.override { go = go_1_26; }) {
  pname = "kubectl-ondemand";
  inherit version;

  # Only the files needed to build the binary, so unrelated changes
  # (docs, CI config, the flox env) don't invalidate the build.
  src = lib.fileset.toSource {
    root = ../.;
    fileset = lib.fileset.unions [
      ../go.mod
      ../go.sum
      ../cmd
      ../internal
    ];
  };

  vendorHash = "sha256-JyrEXuI9B1dPhhAtzpiz4s7+T4yUaRBH74PKky9Q4JI=";

  subPackages = [ "cmd/kubectl-ondemand" ];

  env.CGO_ENABLED = 0;

  ldflags = [
    "-s"
    "-w"
    "-X main.version=${version}"
  ];

  # Sanity-check that the built binary runs and reports its version.
  nativeBuildInputs = [ versionCheckHook ];
  doInstallCheck = true;

  meta = {
    description = "kubectl/krew plugin that analyzes why Karpenter nodes are on-demand";
    longDescription = ''
      Analyzes on-demand nodes managed by Karpenter and classifies why
      each node and its workloads are running on on-demand capacity instead
      of spot. Identifies requested, spot-fallback, and inherited cases,
      and highlights misconfigured workloads that could run on spot.

      Features:
      - Classifies on-demand nodes as requested, spot-fallback, or inherited
      - Shows per-pod analysis of why workloads are on on-demand nodes
      - Detects misconfigured workloads that could run on spot
      - Calculates spot-capable percentage per node
      - Configurable spot taint validation
      - Automatically detects Karpenter API version (v1alpha5, v1beta1, v1)
      - Supports table, JSON, and YAML output formats

      Because kubectl discovers `kubectl-*` executables on PATH as plugins,
      installing this package makes `kubectl ondemand` available.
    '';
    homepage = "https://github.com/ssoriche/kubectl-ondemand";
    license = lib.licenses.mit;
    mainProgram = "kubectl-ondemand";
    maintainers = [ ];
  };
}
