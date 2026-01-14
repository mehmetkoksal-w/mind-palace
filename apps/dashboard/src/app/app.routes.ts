import { Routes } from "@angular/router";

export const routes: Routes = [
  { path: "", redirectTo: "/overview", pathMatch: "full" },
  {
    path: "onboarding",
    loadComponent: () =>
      import("./features/onboarding/onboarding.component").then(
        (m) => m.OnboardingComponent
      ),
  },
  {
    path: "overview",
    loadComponent: () =>
      import("./features/overview/overview.component").then(
        (m) => m.OverviewComponent
      ),
  },
  {
    path: "explore",
    loadComponent: () =>
      import("./features/explore/explore.component").then(
        (m) => m.ExploreComponent
      ),
    children: [
      { path: "", redirectTo: "rooms", pathMatch: "full" },
      {
        path: "rooms",
        loadComponent: () =>
          import("./features/rooms/rooms.component").then(
            (m) => m.RoomsComponent
          ),
      },
      {
        path: "graph",
        loadComponent: () =>
          import("./features/graph/graph.component").then(
            (m) => m.GraphComponent
          ),
      },
      {
        path: "intel",
        loadComponent: () =>
          import("./features/intel/intel.component").then(
            (m) => m.IntelComponent
          ),
      },
    ],
  },
  {
    path: "insights",
    loadComponent: () =>
      import("./features/insights/insights.component").then(
        (m) => m.InsightsComponent
      ),
    children: [
      { path: "", redirectTo: "sessions", pathMatch: "full" },
      {
        path: "sessions",
        loadComponent: () =>
          import("./features/sessions/sessions.component").then(
            (m) => m.SessionsComponent
          ),
      },
      {
        path: "learnings",
        loadComponent: () =>
          import("./features/learnings/learnings.component").then(
            (m) => m.LearningsComponent
          ),
      },
      {
        path: "ideas",
        loadComponent: () =>
          import("./features/ideas/ideas.component").then(
            (m) => m.IdeasComponent
          ),
      },
      {
        path: "decisions",
        loadComponent: () =>
          import("./features/decisions/decisions.component").then(
            (m) => m.DecisionsComponent
          ),
      },
      {
        path: "corridors",
        loadComponent: () =>
          import("./features/corridors/corridors.component").then(
            (m) => m.CorridorsComponent
          ),
      },
      {
        path: "conversations",
        loadComponent: () =>
          import("./features/conversations/conversations.component").then(
            (m) => m.ConversationsComponent
          ),
      },
      {
        path: "contradictions",
        loadComponent: () =>
          import("./features/contradictions/contradictions.component").then(
            (m) => m.ContradictionsComponent
          ),
      },
      {
        path: "postmortems",
        loadComponent: () =>
          import("./features/postmortems/postmortems.component").then(
            (m) => m.PostmortemsComponent
          ),
      },
      {
        path: "proposals",
        loadComponent: () =>
          import("./features/proposals/proposals.component").then(
            (m) => m.ProposalsComponent
          ),
      },
      {
        path: "decision-timeline",
        loadComponent: () =>
          import(
            "./features/decision-timeline/decision-timeline.component"
          ).then((m) => m.DecisionTimelineComponent),
      },
      {
        path: "context-preview",
        loadComponent: () =>
          import("./features/context-preview/context-preview.component").then(
            (m) => m.ContextPreviewComponent
          ),
      },
      {
        path: "scope-explorer",
        loadComponent: () =>
          import("./features/scope-explorer/scope-explorer.component").then(
            (m) => m.ScopeExplorerComponent
          ),
      },
    ],
  },
  // Fallback redirect for old routes
  { path: "rooms", redirectTo: "/explore/rooms", pathMatch: "full" },
  { path: "graph", redirectTo: "/explore/graph", pathMatch: "full" },
  { path: "intel", redirectTo: "/explore/intel", pathMatch: "full" },
  { path: "sessions", redirectTo: "/insights/sessions", pathMatch: "full" },
  { path: "learnings", redirectTo: "/insights/learnings", pathMatch: "full" },
  { path: "corridors", redirectTo: "/insights/corridors", pathMatch: "full" },
  // Catch all
  { path: "**", redirectTo: "/overview" },
];
