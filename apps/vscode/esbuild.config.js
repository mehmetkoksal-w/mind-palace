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

// Blueprint webview bundle (runs in browser)
const blueprintWebviewConfig = {
  entryPoints: ["src/webviews/webview-scripts/blueprint.ts"],
  bundle: true,
  outfile: "out/webviews/blueprint.js",
  format: "iife", // Immediately Invoked Function Expression
  platform: "browser",
  sourcemap: true,
  minify: production,
  target: ["es2020"],
};

// Knowledge graph webview bundle (runs in browser)
const knowledgeGraphWebviewConfig = {
  entryPoints: ["src/webviews/webview-scripts/knowledge-graph.ts"],
  bundle: true,
  outfile: "out/webviews/knowledge-graph.js",
  format: "iife",
  platform: "browser",
  sourcemap: true,
  minify: production,
  target: ["es2020"],
};

async function build() {
  try {
    if (watch) {
      // Watch mode for development
      const contexts = await Promise.all([
        esbuild.context(extensionConfig),
        esbuild.context(blueprintWebviewConfig),
        esbuild.context(knowledgeGraphWebviewConfig),
      ]);
      await Promise.all(contexts.map((ctx) => ctx.watch()));
      console.log("[esbuild] Watching for changes...");
    } else {
      // Production build
      await Promise.all([
        esbuild.build(extensionConfig),
        esbuild.build(blueprintWebviewConfig),
        esbuild.build(knowledgeGraphWebviewConfig),
      ]);
      console.log("[esbuild] Build complete");
    }
  } catch (error) {
    console.error("[esbuild] Build failed:", error);
    process.exit(1);
  }
}

build();
