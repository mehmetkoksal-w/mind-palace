const esbuild = require("esbuild");

const watch = process.argv.includes("--watch");
const production = process.env.NODE_ENV === "production";

// Extension bundle (runs in Node.js)
const extensionConfig = {
  entryPoints: ["src/extension.ts"],
  bundle: true,
  outfile: "out/extension.js",
  external: ["vscode"],
  format: "cjs",
  platform: "node",
  sourcemap: true,
  minify: production,
};

async function build() {
  try {
    if (watch) {
      // Watch mode for development
      const ctx = await esbuild.context(extensionConfig);
      await ctx.watch();
      console.log("[esbuild] Watching for changes...");
    } else {
      // Production build
      await esbuild.build(extensionConfig);
      console.log("[esbuild] Build complete");
    }
  } catch (error) {
    console.error("[esbuild] Build failed:", error);
    process.exit(1);
  }
}

build();
