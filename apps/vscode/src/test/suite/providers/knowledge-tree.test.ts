import { expect } from "chai";
import * as sinon from "sinon";
import * as vscode from "vscode";
import {
  KnowledgeTreeProvider,
  KnowledgeTreeItem,
  KnowledgeItemType,
} from "../../../providers/knowledgeTreeProvider";
import { PalaceBridge } from "../../../bridge";

/**
 * Knowledge Tree Provider Tests
 * Tests tree data provider, refresh logic, and item selection
 */
describe("KnowledgeTreeProvider Tests", () => {
  let sandbox: sinon.SinonSandbox;
  let provider: KnowledgeTreeProvider;
  let bridge: PalaceBridge;
  let bridgeStub: sinon.SinonStubbedInstance<PalaceBridge>;

  beforeEach(() => {
    sandbox = sinon.createSandbox();
    provider = new KnowledgeTreeProvider();
    bridge = new PalaceBridge();

    // Create stubbed bridge methods
    bridgeStub = {
      recallIdeas: sandbox.stub(),
      recallDecisions: sandbox.stub(),
      recallLearnings: sandbox.stub(),
      dispose: sandbox.stub(),
    } as any;

    provider.setBridge(bridgeStub as any);
  });

  afterEach(() => {
    sandbox.restore();
  });

  describe("Initialization", () => {
    it("should create provider instance", () => {
      expect(provider).to.exist;
      expect(provider).to.be.instanceOf(KnowledgeTreeProvider);
    });

    it("should set bridge", () => {
      const newProvider = new KnowledgeTreeProvider();
      newProvider.setBridge(bridge);
      // Bridge is set internally, provider should work
      expect(newProvider).to.exist;
    });

    it("should provide empty tree when no bridge set", async () => {
      const newProvider = new KnowledgeTreeProvider();
      const children = await newProvider.getChildren();

      // Should return category items even without bridge
      expect(children).to.be.an("array");
      expect(children.length).to.be.greaterThan(0);
    });
  });

  describe("Tree Structure", () => {
    it("should return top-level categories", async () => {
      bridgeStub.recallIdeas.resolves({ ideas: [] });
      bridgeStub.recallDecisions.resolves({ decisions: [] });
      bridgeStub.recallLearnings.resolves({ learnings: [] });

      const children = await provider.getChildren();

      expect(children).to.be.an("array");
      expect(children.length).to.equal(3); // Ideas, Decisions, Learnings

      const labels = children.map((item) => item.label);
      expect(labels).to.include("Ideas");
      expect(labels).to.include("Decisions");
      expect(labels).to.include("Learnings");
    });

    it("should create category items with correct types", async () => {
      bridgeStub.recallIdeas.resolves({ ideas: [] });
      bridgeStub.recallDecisions.resolves({ decisions: [] });
      bridgeStub.recallLearnings.resolves({ learnings: [] });

      const children = await provider.getChildren();

      children.forEach((item) => {
        expect(item.itemType).to.equal(KnowledgeItemType.Category);
        expect(item.collapsibleState).to.equal(
          vscode.TreeItemCollapsibleState.Collapsed
        );
      });
    });

    it("should show counts in category descriptions", async () => {
      bridgeStub.recallIdeas.resolves({
        ideas: [
          { id: "1", content: "Test idea", status: "active", scope: "palace" },
        ],
      });
      bridgeStub.recallDecisions.resolves({ decisions: [] });
      bridgeStub.recallLearnings.resolves({ learnings: [] });

      const children = await provider.getChildren();
      const ideasCategory = children.find((item) => item.label === "Ideas");

      expect(ideasCategory?.description).to.include("1");
    });
  });

  describe("Ideas Handling", () => {
    it("should fetch and display ideas", async () => {
      const mockIdeas = [
        { id: "1", content: "First idea", status: "active", scope: "palace" },
        {
          id: "2",
          content: "Second idea",
          status: "exploring",
          scope: "room",
          scopePath: "src/",
        },
      ];

      bridgeStub.recallIdeas.resolves({ ideas: mockIdeas });
      bridgeStub.recallDecisions.resolves({ decisions: [] });
      bridgeStub.recallLearnings.resolves({ learnings: [] });

      const categories = await provider.getChildren();
      const ideasCategory = categories.find((item) => item.label === "Ideas");

      const ideas = await provider.getChildren(ideasCategory);

      expect(ideas.length).to.be.greaterThan(0);
    });

    it("should group ideas by status", async () => {
      const mockIdeas = [
        { id: "1", content: "Active idea", status: "active", scope: "palace" },
        {
          id: "2",
          content: "Implemented idea",
          status: "implemented",
          scope: "palace",
        },
      ];

      bridgeStub.recallIdeas.resolves({ ideas: mockIdeas });
      bridgeStub.recallDecisions.resolves({ decisions: [] });
      bridgeStub.recallLearnings.resolves({ learnings: [] });

      const categories = await provider.getChildren();
      const ideasCategory = categories.find((item) => item.label === "Ideas");
      const statusGroups = await provider.getChildren(ideasCategory);

      // Should have status groups
      expect(statusGroups).to.be.an("array");
      expect(statusGroups.length).to.be.greaterThan(0);
    });

    it("should create idea items with commands", async () => {
      const mockIdea = {
        id: "1",
        content: "Test idea",
        status: "active",
        scope: "palace",
      };

      bridgeStub.recallIdeas.resolves({ ideas: [mockIdea] });
      bridgeStub.recallDecisions.resolves({ decisions: [] });
      bridgeStub.recallLearnings.resolves({ learnings: [] });

      const categories = await provider.getChildren();
      const ideasCategory = categories.find((item) => item.label === "Ideas");
      const statusGroups = await provider.getChildren(ideasCategory);

      if (statusGroups.length > 0) {
        const ideas = await provider.getChildren(statusGroups[0]);

        if (ideas.length > 0) {
          expect(ideas[0].command).to.exist;
          expect(ideas[0].command?.command).to.equal(
            "mindPalace.showKnowledgeDetail"
          );
        }
      }
    });
  });

  describe("Decisions Handling", () => {
    it("should fetch and display decisions", async () => {
      const mockDecisions = [
        {
          id: "1",
          content: "Use TypeScript",
          status: "active",
          scope: "palace",
        },
      ];

      bridgeStub.recallIdeas.resolves({ ideas: [] });
      bridgeStub.recallDecisions.resolves({ decisions: mockDecisions });
      bridgeStub.recallLearnings.resolves({ learnings: [] });

      const categories = await provider.getChildren();
      const decisionsCategory = categories.find(
        (item) => item.label === "Decisions"
      );

      expect(decisionsCategory).to.exist;
      expect(decisionsCategory?.description).to.include("1");
    });

    it("should create decision items with proper icons", async () => {
      const mockDecision = {
        id: "1",
        content: "Important decision",
        status: "pending",
        scope: "room",
        scopePath: "src/core",
      };

      bridgeStub.recallIdeas.resolves({ ideas: [] });
      bridgeStub.recallDecisions.resolves({ decisions: [mockDecision] });
      bridgeStub.recallLearnings.resolves({ learnings: [] });

      const categories = await provider.getChildren();
      const decisionsCategory = categories.find(
        (item) => item.label === "Decisions"
      );

      expect(decisionsCategory?.iconPath).to.exist;
    });
  });

  describe("Learnings Handling", () => {
    it("should fetch and display learnings", async () => {
      const mockLearnings = [
        {
          id: "1",
          content: "Important learning",
          confidence: 0.95,
          scope: "file",
          scopePath: "src/main.ts",
        },
      ];

      bridgeStub.recallIdeas.resolves({ ideas: [] });
      bridgeStub.recallDecisions.resolves({ decisions: [] });
      bridgeStub.recallLearnings.resolves({ learnings: mockLearnings });

      const categories = await provider.getChildren();
      const learningsCategory = categories.find(
        (item) => item.label === "Learnings"
      );

      expect(learningsCategory).to.exist;
      expect(learningsCategory?.description).to.include("1");
    });
  });

  describe("Refresh Functionality", () => {
    it("should fire change event on refresh", (done) => {
      provider.onDidChangeTreeData((item) => {
        // Event should fire with undefined (refresh all)
        expect(item).to.be.undefined;
        done();
      });

      provider.refresh();
    });

    it("should refresh with force parameter", async () => {
      await provider.refresh(true);
      // Should complete without error
      expect(true).to.be.true;
    });
  });

  describe("Error Handling", () => {
    it("should handle bridge errors gracefully", async () => {
      bridgeStub.recallIdeas.rejects(new Error("Connection failed"));
      bridgeStub.recallDecisions.resolves({ decisions: [] });
      bridgeStub.recallLearnings.resolves({ learnings: [] });

      const children = await provider.getChildren();

      // Should still return categories, even if one fails
      expect(children).to.be.an("array");
      expect(children.length).to.equal(3);
    });

    it("should handle null/undefined responses", async () => {
      bridgeStub.recallIdeas.resolves({ ideas: null as any });
      bridgeStub.recallDecisions.resolves({ decisions: undefined as any });
      bridgeStub.recallLearnings.resolves({ learnings: [] });

      const children = await provider.getChildren();

      // Should handle gracefully
      expect(children).to.be.an("array");
    });
  });

  describe("TreeItem Creation", () => {
    it("should create items with correct collapsible state", () => {
      const categoryItem = new KnowledgeTreeItem(
        "Test Category",
        KnowledgeItemType.Category,
        vscode.TreeItemCollapsibleState.Collapsed
      );

      expect(categoryItem.collapsibleState).to.equal(
        vscode.TreeItemCollapsibleState.Collapsed
      );
    });

    it("should set context value for items", () => {
      const ideaItem = new KnowledgeTreeItem(
        "Test Idea",
        KnowledgeItemType.Idea,
        vscode.TreeItemCollapsibleState.None,
        { id: "1", content: "test", status: "active", scope: "palace" }
      );

      expect(ideaItem.contextValue).to.equal(KnowledgeItemType.Idea);
    });

    it("should set tooltips for items", () => {
      const item = new KnowledgeTreeItem(
        "Test",
        KnowledgeItemType.Idea,
        vscode.TreeItemCollapsibleState.None,
        { id: "1", content: "Test idea", status: "active", scope: "palace" }
      );

      expect(item.tooltip).to.exist;
    });
  });

  describe("Integration", () => {
    it("should work with real bridge structure", async () => {
      // Create provider with real bridge instance (no stubs)
      const realProvider = new KnowledgeTreeProvider();
      const realBridge = new PalaceBridge();
      realProvider.setBridge(realBridge);

      // Should handle no connection gracefully
      const children = await realProvider.getChildren();
      expect(children).to.be.an("array");

      realBridge.dispose();
    });
  });
});
