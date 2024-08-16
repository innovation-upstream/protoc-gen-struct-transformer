{ pkgs ? import <nixpkgs> {}, ... }:

pkgs.mkShell {
  buildInputs = with pkgs; [
    buf
    protoc-gen-go
    go_1_22
    bazel
  ];
}
