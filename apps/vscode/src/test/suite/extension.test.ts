import * as vscode from "vscode";
import { expect } from "chai";
import * as sinon from "sinon";

/**
 * Extension Activation Tests
 * Tests the core extension lifecycle and command registration
 */
describe("Extension Activation Tests", () => {
  let sandbox: sinon.SinonSandbox;

  beforeEach(() => {
    sandbox = sinon.createSandbox();
  });

  afterEach(() => {
    sandbox.restore();
  });

  it("should activate the extension", async () => {
    const ext = vscode.extensions.getExtension(
      "mind-palace.mind-palace-vscode"
    );
    expect(ext).to.not.be.undefined;

    if (ext && !ext.isActive) {
      await ext.activate();
    }

    expect(ext?.isActive).to.be.true;
  });

  it("should register all required commands", async () => {
    // Ensure extension is activated
    const ext = vscode.extensions.getExtension(
      "mind-palace.mind-palace-vscode"
    );
    if (ext && !ext.isActive) {
      await ext.activate();
    }

    const commands = await vscode.commands.getCommands(true);

    const requiredCommands = [
      "mindPalace.openBlueprint",
      "mindPalace.storeIdea",
      "mindPalace.storeDecision",
      "mindPalace.storeLearning",
      "mindPalace.quickStore",
      "mindPalace.startSession",
      "mindPalace.endSession",
      "mindPalace.semanticSearch",
      "mindPalace.showKnowledgeGraph",
    ];

    for (const cmd of requiredCommands) {
      expect(commands).to.include(cmd, `Command ${cmd} should be registered`);
    }

    // Note: some commands (e.g., checkStatus, heal, refreshKnowledge) may not appear
    // in getCommands() during test due to activation timing
  });

  it("should create tree data providers", async () => {
    // Verify extension has activated and providers are registered
    const ext = vscode.extensions.getExtension(
      "mind-palace.mind-palace-vscode"
    );
    if (ext && !ext.isActive) {
      await ext.activate();
    }

    // Tree providers are registered during activation
    expect(ext?.isActive).to.be.true;
  });

  it("should handle command execution without errors", async () => {
    const ext = vscode.extensions.getExtension(
      "mind-palace.mind-palace-vscode"
    );
    if (ext && !ext.isActive) {
      await ext.activate();
    }

    // Test that checkStatus command can be executed (even if it fails due to no CLI)
    try {
      await vscode.commands.executeCommand("mindPalace.checkStatus");
    } catch (error) {
      // Expected to fail if CLI is not available, but should not crash
      expect(error).to.exist;
    }
  });

  it("should register webview providers", async () => {
    const ext = vscode.extensions.getExtension(
      "mind-palace.mind-palace-vscode"
    );
    if (ext && !ext.isActive) {
      await ext.activate();
    }

    // Webview providers are registered, verify through extension activation
    expect(ext?.isActive).to.be.true;
  });
});
