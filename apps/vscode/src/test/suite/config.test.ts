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
      // Stub workspace methods
      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: sandbox.stub().returns(undefined),
      } as any);

      sandbox.stub(vscode.workspace, "workspaceFolders").value(undefined);

      const config = getConfig();

      expect(config.autoSync).to.equal(true);
      expect(config.autoSyncDelay).to.equal(1500);
      expect(config.waitForCleanWorkspace).to.equal(false);
      expect(config.decorations.enabled).to.equal(true);
      expect(config.decorations.style).to.equal("gutter");
      expect(config.statusBar.position).to.equal("left");
      expect(config.sidebar.defaultView).to.equal("tree");
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

    it("should read autoSync from VS Code settings", () => {
      const getStub = sandbox.stub();
      getStub.withArgs("autoSync").returns(false);
      getStub.returns(undefined);

      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: getStub,
      } as any);

      const config = getConfig();
      expect(config.autoSync).to.equal(false);
    });

    it("should read autoSyncDelay from VS Code settings", () => {
      const getStub = sandbox.stub();
      getStub.withArgs("autoSyncDelay").returns(3000);
      getStub.returns(undefined);

      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: getStub,
      } as any);

      const config = getConfig();
      expect(config.autoSyncDelay).to.equal(3000);
    });

    it("should read waitForCleanWorkspace from VS Code settings", () => {
      const getStub = sandbox.stub();
      getStub.withArgs("waitForCleanWorkspace").returns(true);
      getStub.returns(undefined);

      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: getStub,
      } as any);

      const config = getConfig();
      expect(config.waitForCleanWorkspace).to.equal(true);
    });
  });

  describe("Project Configuration (.palace/palace.jsonc)", () => {
    it("should merge project config with defaults", () => {
      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: sandbox.stub().returns(undefined),
      } as any);

      // Mock project config file
      const mockProjectConfig = {
        vscode: {
          autoSync: false,
          autoSyncDelay: 2500,
          decorations: {
            enabled: false,
            style: "inline",
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

      expect(config.autoSync).to.equal(false);
      expect(config.autoSyncDelay).to.equal(2500);
      expect(config.decorations.enabled).to.equal(false);
      expect(config.decorations.style).to.equal("inline");
    });

    it("should prioritize project config over VS Code settings", () => {
      const getStub = sandbox.stub();
      getStub.withArgs("autoSync").returns(true);
      getStub.returns(undefined);

      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: getStub,
      } as any);

      const mockProjectConfig = {
        vscode: {
          autoSync: false,
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
      expect(config.autoSync).to.equal(false); // Project config wins
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
      expect(config.autoSync).to.equal(true); // Falls back to defaults
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
      expect(config.autoSync).to.equal(true); // Falls back to defaults
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

    it("should trigger callback on config change", () => {
      const mockUri = {
        fsPath: "/test/workspace",
        scheme: "file",
        path: "/test/workspace",
      };
      sandbox
        .stub(vscode.workspace, "workspaceFolders")
        .value([{ uri: mockUri, name: "test", index: 0 }]);

      let changeHandler: any;
      const mockWatcher = {
        onDidChange: (handler: any) => {
          changeHandler = handler;
          return { dispose: sandbox.stub() };
        },
        onDidCreate: sandbox.stub().returns({ dispose: sandbox.stub() }),
        onDidDelete: sandbox.stub().returns({ dispose: sandbox.stub() }),
        dispose: sandbox.stub(),
      };

      sandbox
        .stub(vscode.workspace, "createFileSystemWatcher")
        .returns(mockWatcher as any);
      sandbox
        .stub(vscode.workspace, "workspaceFolders")
        .value([{ uri: { fsPath: "/test/workspace" } }]);

      const callback = sandbox.stub();
      watchProjectConfig(callback);

      // Trigger change
      if (changeHandler) {
        changeHandler();
        expect(callback.called).to.be.true;
      }
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
    it("should have correct decoration config structure", () => {
      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: sandbox.stub().returns(undefined),
      } as any);

      const config = getConfig();

      expect(config.decorations).to.exist;
      expect(config.decorations.enabled).to.be.a("boolean");
      expect(config.decorations.style).to.be.oneOf([
        "gutter",
        "inline",
        "both",
      ]);
    });

    it("should have correct statusBar config structure", () => {
      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: sandbox.stub().returns(undefined),
      } as any);

      const config = getConfig();

      expect(config.statusBar).to.exist;
      expect(config.statusBar.position).to.be.oneOf(["left", "right"]);
      expect(config.statusBar.priority).to.be.a("number");
    });

    it("should have correct sidebar config structure", () => {
      sandbox.stub(vscode.workspace, "getConfiguration").returns({
        get: sandbox.stub().returns(undefined),
      } as any);

      const config = getConfig();

      expect(config.sidebar).to.exist;
      expect(config.sidebar.defaultView).to.be.oneOf(["tree", "graph"]);
      expect(config.sidebar.graphLayout).to.be.oneOf([
        "cose",
        "circle",
        "grid",
        "breadthfirst",
      ]);
    });
  });
});
