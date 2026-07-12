import type { components } from '@/contracts/openapi/generated/schema';

export type SecurityOverviewResponse = components['schemas']['SecurityOverviewResponse'];
export type SecurityOverviewQuery = {
  preset?: SecurityOverviewResponse['time_preset'];
};
