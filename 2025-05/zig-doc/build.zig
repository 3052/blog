const std = @import("std");

pub fn build(b: *std.Build) void {
   const lib = b.addObject(.{
      .name = "protobuf-zig",
      .root_module = b.createModule(.{
         .root_source_file = b.dependency("protobuf_zig", .{}).module("protobuf-zig").root_source_file,
         .target = b.standardTargetOptions(.{})
      })
   });
   const install_docs = b.addInstallDirectory(.{
      .install_dir = .prefix,
      .install_subdir = "docs",
      .source_dir = lib.getEmittedDocs()
   });
   const docs_step = b.step("docs", "Install docs into zig-out/docs");
   docs_step.dependOn(&install_docs.step);
}

