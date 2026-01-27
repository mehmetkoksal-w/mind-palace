import { expect } from "chai";
import * as sinon from "sinon";
import * as vscode from "vscode";
import {
  getConfig,
  watchProjectConfig,
  fsAdapter,
} from "../../config";

/**
 * Configuration Tests
 * Tests config reading, palace.jsonc merging, and default values
 */
describe("Configuration Tests", () => {
  let sandbox: sinon.SinonSandbox;

  beforeEach(() => {
    sandbox = sinon.createSandbox();
  });

  afterEach(() => {
    sandbox.restore();
  });

  describe("Default Configuration", () => {
    it("should return default values when no config exists", () => {
      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: sandbox.stub().returns(undefined),
      } as any);

      sandbox.stub(vscode.workspace, "workspaceFolders").value(undefined);

      const config = getConfig();

      expect(config.showStatusBarItem).to.equal(true);
      expect(config.lsp.enabled).to.equal(true);
      expect(config.lsp.diagnostics.patterns).to.equal(true);
      expect(config.lsp.diagnostics.contracts).to.equal(true);
      expect(config.lsp.codeLens.enabled).to.equal(true);
      expect(config.statusBar.position).to.equal("left");
      expect(config.statusBar.priority).to.equal(100);
    });

    it('should use default binary path "palace"', () => {
      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: sandbox.stub().returns(undefined),
      } as any);

      const config = getConfig();
      expect(config.binaryPath).to.equal("palace");
    });
  });

  describe("VS Code Settings", () => {
    it("should read binaryPath from VS Code settings", () => {
      const getStub = sandbox.stub();
      getStub.withArgs("binaryPath").returns("/custom/path/to/palace");
      getStub.returns(undefined);

      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: getStub,
      } as any);

      const config = getConfig();
      expect(config.binaryPath).to.equal("/custom/path/to/palace");
    });

    it("should read showStatusBarItem from VS Code settings", () => {
      const getStub = sandbox.stub();
      getStub.withArgs("showStatusBarItem").returns(false);
      getStub.returns(undefined);

      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: getStub,
      } as any);

      const config = getConfig();
      expect(config.showStatusBarItem).to.equal(false);
    });

    it("should read lsp.enabled from VS Code settings", () => {
      const getStub = sandbox.stub();
      getStub.withArgs("lsp.enabled").returns(false);
      getStub.returns(undefined);

      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: getStub,
      } as any);

      const config = getConfig();
      expect(config.lsp.enabled).to.equal(false);
    });
  });

  describe("Project Configuration (.palace/palace.jsonc)", () => {
    it("should merge project config with defaults", () => {
      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: sandbox.stub().returns(undefined),
      } as any);

      const mockProjectConfig = {
        vscode: {
          statusBar: {
            position: "right",
            priority: 50,
          },
        },
      };

      sandbox
        .stub(vscode.workspace, "workspaceFolders")
        .value([{ uri: { fsPath: "/test/workspace" } }]);

      sandbox
        .stub(fsAdapter, "readFileSync")
        .returns(JSON.stringify(mockProjectConfig));
      sandbox.stub(fsAdapter, "existsSync").returns(true);

      const config = getConfig();

      expect(config.statusBar.position).to.equal("right");
      expect(config.statusBar.priority).to.equal(50);
    });

    it("should handle missing palace.jsonc gracefully", () => {
      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: sandbox.stub().returns(undefined),
      } as any);

      sandbox
        .stub(vscode.workspace, "workspaceFolders")
        .value([{ uri: { fsPath: "/test/workspace" } }]);

      sandbox.stub(fsAdapter, "existsSync").returns(false);

      const config = getConfig();
      expect(config.showStatusBarItem).to.equal(true); // Falls back to defaults
    });

    it("should handle malformed palace.jsonc gracefully", () => {
      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: sandbox.stub().returns(undefined),
      } as any);

      sandbox
        .stub(vscode.workspace, "workspaceFolders")
        .value([{ uri: { fsPath: "/test/workspace" } }]);

      sandbox.stub(fsAdapter, "existsSync").returns(true);
      sandbox.stub(fsAdapter, "readFileSync").returns("{ invalid json }");

      const config = getConfig();
      expect(config.showStatusBarItem).to.equal(true); // Falls back to defaults
    });
  });

  describe("Config Watcher", () => {
    it("should create a file watcher", () => {
      const createFileSystemWatcherStub = sandbox.stub(
        vscode.workspace,
        "createFileSystemWatcher"
      );
      const mockWatcher = {
        onDidChange: sandbox.stub().returns({ dispose: sandbox.stub() }),
        onDidCreate: sandbox.stub().returns({ dispose: sandbox.stub() }),
        onDidDelete: sandbox.stub().returns({ dispose: sandbox.stub() }),
        dispose: sandbox.stub(),
      };
      createFileSystemWatcherStub.returns(mockWatcher as any);

      const mockUri = {
        fsPath: "/test/workspace",
        scheme: "file",
        path: "/test/workspace",
      };
      sandbox
        .stub(vscode.workspace, "workspaceFolders")
        .value([{ uri: mockUri, name: "test", index: 0 }]);

      const callback = sandbox.stub();
      const watcher = watchProjectConfig(callback);

      expect(createFileSystemWatcherStub.called).to.be.true;
      expect(watcher).to.exist;
    });

    it("should handle no workspace folders", () => {
      sandbox.stub(vscode.workspace, "workspaceFolders").value(undefined);

      const callback = sandbox.stub();
      const watcher = watchProjectConfig(callback);

      expect(watcher).to.exist;
      watcher.dispose();
    });
  });

  describe("Configuration Structure", () => {
    it("should have correct statusBar config structure", () => {
      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: sandbox.stub().returns(undefined),
      } as any);

      const config = getConfig();

      expect(config.statusBar).to.exist;
      expect(config.statusBar.position).to.be.oneOf(["left", "right"]);
      expect(config.statusBar.priority).to.be.a("number");
    });

    it("should have correct lsp config structure", () => {
      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: sandbox.stub().returns(undefined),
      } as any);

      const config = getConfig();

      expect(config.lsp).to.exist;
      expect(config.lsp.enabled).to.be.a("boolean");
      expect(config.lsp.diagnostics.patterns).to.be.a("boolean");
      expect(config.lsp.diagnostics.contracts).to.be.a("boolean");
      expect(config.lsp.codeLens.enabled).to.be.a("boolean");
    });
  });
});
