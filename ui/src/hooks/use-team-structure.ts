import { useEffect, useMemo, useReducer } from "react";
import type { Node, Edge } from "@xyflow/react";
import { api } from "@/api";
import { logger } from "@/lib/logger";
import { getErrorMessage } from "@/lib/utilities";
import { teamLayout, type StructureMember } from "@/components/team-structure/team-layout";

// --- Shared types --

interface StructureData {
  rootTeamMemberId: string;
  members: Array<StructureMember & { cliProfileName: string; systemPromptNames: string[] }>;
  relations: Array<{ from: string; to: string }>;
}

const EMPTY_LAYOUT = { nodes: [] as Node[], edges: [] as Edge[] };

// --- Shared layout helper --

function buildLayout(structure: StructureData | undefined) {
  if (!structure) return EMPTY_LAYOUT;

  const cliProfileNames = new Map(structure.members.map((m) => [m.cliProfileId, m.cliProfileName]));
  const systemPromptNames = new Map(
    structure.members.flatMap((m) => m.systemPromptIds.map((id, i) => [id, m.systemPromptNames[i]] as const)),
  );

  return teamLayout(structure.members, structure.relations, {
    rootId: structure.rootTeamMemberId,
    cliProfileNames,
    systemPromptNames,
  });
}

// --- Reducer (used by useTeamStructure only) --

interface State {
  structure: StructureData | undefined;
  error: string | undefined;
}

type Action =
  | { type: "reset" }
  | { type: "loaded"; structure: StructureData }
  | { type: "errored"; message: string };

const INITIAL_STATE: State = {
  structure: undefined,
  error: undefined,
};

function reducer(_state: State, action: Action): State {
  switch (action.type) {
    case "reset": {
      return INITIAL_STATE;
    }
    case "loaded": {
      return { structure: action.structure, error: undefined };
    }
    case "errored": {
      return { ...INITIAL_STATE, error: action.message };
    }
  }
}

// --- Hooks --

/**
 * Fetches live team structure from the API and produces graph nodes/edges.
 * Used on the team-detail page.
 */
export function useTeamStructure(teamId: string | undefined) {
  const [state, dispatch] = useReducer(reducer, INITIAL_STATE);

  useEffect(() => {
    if (!teamId) return;

    dispatch({ type: "reset" });

    api.teams
      .getStructure(teamId)
      .then((data) => {
        dispatch({ type: "loaded", structure: data });
      })
      .catch((error_: unknown) => {
        logger.error("Failed to load team structure", { error: error_ });
        dispatch({ type: "errored", message: getErrorMessage(error_, "Failed to load") });
      });
  }, [teamId]);

  const layout = useMemo(() => buildLayout(state.structure), [state.structure]);
  const ready = state.structure !== undefined || state.error !== undefined;

  return {
    nodes: layout.nodes,
    edges: layout.edges,
    ready,
    empty: state.structure !== undefined && layout.nodes.length === 0,
    error: state.error,
  };
}
