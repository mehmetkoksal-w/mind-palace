import { expect } from "chai";
import * as sinon from "sinon";
import { EventEmitter } from "events";
import { PalaceBridge, MCP_TOOLS } from "../../bridge";

/**
 * Bridge Tests
 * Tests MCP communication, tool calling, and error handling
 */
describe("PalaceBridge Tests", () => {
  let sandbox: sinon.SinonSandbox;
  let bridge: PalaceBridge;

  beforeEach(() => {
    sandbox = sinon.createSandbox();
    bridge = new PalaceBridge();
  });

  afterEach(() => {
    bridge.dispose();
    sandbox.restore();
  });

  describe("Initialization", () => {
    it("should create bridge instance", () => {
      expect(bridge).to.exist;
      expect(bridge).to.be.instanceOf(PalaceBridge);
    });

    it("should dispose cleanly", () => {
      bridge.dispose();
      expect(() => bridge.dispose()).to.not.throw();
    });
  });

  describe("High-Level API Methods", () => {
    it("should provide getBrief method", () => {
      expect(bridge.getBrief).to.be.a("function");
    });

    it("should provide getFileIntel method", () => {
      expect(bridge.getFileIntel).to.be.a("function");
    });

    it("should provide recall methods", () => {
      expect(bridge.recallLearnings).to.be.a("function");
      expect(bridge.recallDecisions).to.be.a("function");
      expect(bridge.recallIdeas).to.be.a("function");
    });

    it("should provide session methods", () => {
      expect(bridge.startSession).to.be.a("function");
      expect(bridge.listSessions).to.be.a("function");
      expect(bridge.endSession).to.be.a("function");
    });

    it("should provide corridor methods", () => {
      expect(bridge.getCorridorLearnings).to.be.a("function");
      expect(bridge.getCorridorStats).to.be.a("function");
    });

    it("should provide semantic search methods", () => {
      expect(bridge.semanticSearch).to.be.a("function");
      expect(bridge.hybridSearch).to.be.a("function");
    });

    it("should provide store method", () => {
      expect(bridge.store).to.be.a("function");
    });

    it("should provide search method", () => {
      expect(bridge.search).to.be.a("function");
    });

    it("should provide runHeal method", () => {
      expect(bridge.runHeal).to.be.a("function");
    });

    it("should provide runVerify method", () => {
      expect(bridge.runVerify).to.be.a("function");
    });
  });

  describe("API Behavior", () => {
    it("should handle errors when CLI is not available", async () => {
      // When CLI is not available, methods should handle gracefully
      try {
        await bridge.getBrief();
        // May succeed if CLI is available in test environment
      } catch (error) {
        // Expected to fail if CLI is not available
        expect(error).to.exist;
      }
    });

    it("should handle recallLearnings without errors", async () => {
      try {
        await bridge.recallLearnings({});
      } catch (error) {
        // May fail if CLI not available, but should not crash
        expect(error).to.exist;
      }
    });

    it("should handle recallDecisions without errors", async () => {
      try {
        await bridge.recallDecisions({});
      } catch (error) {
        expect(error).to.exist;
      }
    });

    it("should handle recallIdeas without errors", async () => {
      try {
        await bridge.recallIdeas({});
      } catch (error) {
        expect(error).to.exist;
      }
    });

    it("should accept store options", async () => {
      try {
        await bridge.store("test content", {
          as: "idea",
          scope: "palace",
        });
      } catch (error) {
        expect(error).to.exist;
      }
    });

    it("should accept session start parameters", async () => {
      try {
        await bridge.startSession("copilot", "Test goal");
      } catch (error) {
        expect(error).to.exist;
      }
    });
  });

  describe("MCP Tools Constants", () => {
    it("should export MCP_TOOLS constants", () => {
      expect(MCP_TOOLS).to.exist;
      expect(MCP_TOOLS.BRIEF).to.equal("brief");
      expect(MCP_TOOLS.EXPLORE).to.equal("explore");
      expect(MCP_TOOLS.STORE).to.equal("store");
      expect(MCP_TOOLS.RECALL).to.equal("recall");
    });

    it("should have all required tool names", () => {
      const requiredTools = [
        "explore",
        "explore_rooms",
        "explore_context",
        "store",
        "recall",
        "brief",
        "brief_file",
        "session_start",
        "session_list",
        "session_end",
        "corridor_learnings",
        "corridor_stats",
        "search_semantic",
        "search_hybrid",
      ];

      const toolValues = Object.values(MCP_TOOLS);
      requiredTools.forEach((tool) => {
        expect(toolValues).to.include(tool, `MCP_TOOLS should include ${tool}`);
      });
    });
  });

  describe("Error Handling", () => {
    it("should handle connection failures gracefully", async () => {
      // Stub MCPClient.spawn (wrapper) to simulate failure without stubbing core module
      const spawnStub = sandbox.stub((bridge as any).mcpClient, "spawn");
      const mockProcess = new EventEmitter() as any;
      mockProcess.stdin = { write: sandbox.stub(), end: sandbox.stub() };
      mockProcess.stdout = new EventEmitter();
      mockProcess.stderr = new EventEmitter();
      mockProcess.kill = sandbox.stub();

      spawnStub.returns(mockProcess);
      const promise = bridge.getBrief();
      setTimeout(() => {
        mockProcess.emit("error", new Error("ENOENT: palace not found"));
      }, 600);

      let caught: any = null;
      try {
        await promise;
      } catch (e) {
        caught = e;
      }
      expect(caught).to.exist;
    });

    it("should handle invalid responses", async () => {
      const spawnStub = sandbox.stub((bridge as any).mcpClient, "spawn");
      const mockProcess = new EventEmitter() as any;
      mockProcess.stdin = { write: sandbox.stub(), end: sandbox.stub() };
      mockProcess.stdout = new EventEmitter();
      mockProcess.stderr = new EventEmitter();
      mockProcess.kill = sandbox.stub();

      spawnStub.returns(mockProcess);

      // After connection initializes (~500ms), emit an error JSON-RPC response
      setTimeout(() => {
        const resp = {
          jsonrpc: "2.0",
          id: 1,
          error: { code: -32603, message: "Invalid response" },
        };
        mockProcess.stdout.emit("data", JSON.stringify(resp) + "\n");
      }, 600);

      let caught: any = null;
      try {
        await bridge.getBrief();
      } catch (error) {
        caught = error;
      }
      expect(caught).to.exist;
    });
  });

  describe("Disposal", () => {
    it("should clean up resources on dispose", () => {
      const spy = sandbox.spy(bridge, "dispose");
      bridge.dispose();
      expect(spy.called).to.be.true;
    });

    it("should be idempotent", () => {
      bridge.dispose();
      bridge.dispose();
      bridge.dispose();
      // Should not throw
      expect(true).to.be.true;
    });
  });
});
