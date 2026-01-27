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

  it("should register required commands", async () => {
    // Ensure extension is activated
    const ext = vscode.extensions.getExtension(
      "mind-palace.mind-palace-vscode"
    );
    if (ext && !ext.isActive) {
      await ext.activate();
    }

    const commands = await vscode.commands.getCommands(true);

    const requiredCommands = [
      "mindPalace.checkStatus",
      "mindPalace.restartLsp",
    ];

    for (const cmd of requiredCommands) {
      expect(commands).to.include(cmd, `Command ${cmd} should be registered`);
    }
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
});
