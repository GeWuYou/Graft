type ProjectCreated = {
  project_id: number;
};

export type OptionalDeploymentStatus = 'not-requested' | 'succeeded' | 'failed';

export type CreateWithOptionalDeployResult<T extends ProjectCreated> = {
  created: T;
  deployment: {
    status: OptionalDeploymentStatus;
    error?: unknown;
  };
};

/**
 * Runs creation first, then optionally invokes the existing independent deployment action.
 * A deployment failure never invalidates an already-created project.
 */
export async function createWithOptionalDeploy<T extends ProjectCreated>(options: {
  create: () => Promise<T>;
  deploy: (projectId: number) => Promise<unknown>;
  deployAfterCreate: boolean;
}): Promise<CreateWithOptionalDeployResult<T>> {
  const created = await options.create();
  if (!options.deployAfterCreate) {
    return { created, deployment: { status: 'not-requested' } };
  }

  try {
    await options.deploy(created.project_id);
    return { created, deployment: { status: 'succeeded' } };
  } catch (error) {
    return { created, deployment: { status: 'failed', error } };
  }
}
