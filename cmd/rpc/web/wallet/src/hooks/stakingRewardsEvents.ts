export interface RewardEvent {
  eventType: string;
  msg?: {
    amount?: number;
  };
  height: number;
  reference?: string;
  chainId?: number;
  address?: string;
}

interface FetchRewardEventsInRangeParams {
  address: string;
  toHeight: number;
  secondsPerBlock: number;
  hours?: number;
  perPage?: number;
  maxPages?: number;
}

type DsFetch = <T = any>(key: string, ctx?: Record<string, any>) => Promise<T>;

export interface RewardRangeResult {
  events: RewardEvent[];
  fromHeight: number;
  toHeight: number;
  blocksInRange: number;
}

/**
 * Fetch reward events for an address in a height range derived from a time window.
 * The endpoint is paginated, so we iterate until leaving the range or exhausting pages.
 */
export async function fetchRewardEventsInRange(
  dsFetch: DsFetch,
  {
    address,
    toHeight,
    secondsPerBlock,
    hours = 24,
    perPage = 100,
    maxPages = 100,
  }: FetchRewardEventsInRangeParams
): Promise<RewardRangeResult> {
  const safeToHeight = Math.max(0, Number(toHeight) || 0);
  const safeSecondsPerBlock = Math.max(1, Number(secondsPerBlock) || 1);
  const blocksInRange = Math.max(1, Math.round((hours * 60 * 60) / safeSecondsPerBlock));
  const fromHeight = Math.max(0, safeToHeight - blocksInRange);

  if (!address || safeToHeight <= 0) {
    return { events: [], fromHeight, toHeight: safeToHeight, blocksInRange };
  }

  const inRangeEvents: RewardEvent[] = [];

  for (let page = 1; page <= maxPages; page += 1) {
    const pageEvents = await dsFetch<RewardEvent[]>("events.byAddress", {
      address,
      height: safeToHeight,
      page,
      perPage,
    });

    if (!Array.isArray(pageEvents) || pageEvents.length === 0) break;

    const rewards = pageEvents.filter(
      (event) =>
        event?.eventType === "reward" &&
        Number.isFinite(event?.height) &&
        event.height > 0
    );

    const rewardsInRange = rewards.filter(
      (event) => event.height >= fromHeight && event.height <= safeToHeight
    );
    inRangeEvents.push(...rewardsInRange);

    const oldestHeightInPage = rewards.reduce(
      (min, event) => Math.min(min, event.height),
      Number.POSITIVE_INFINITY
    );

    // Data comes newest -> oldest; once oldest in page is below range, following pages are older.
    if (oldestHeightInPage < fromHeight) break;
    if (pageEvents.length < perPage) break;
  }

  return {
    events: inRangeEvents,
    fromHeight,
    toHeight: safeToHeight,
    blocksInRange,
  };
}

export function sumRewards(events: RewardEvent[]): number {
  return events.reduce((sum, event) => sum + (event?.msg?.amount || 0), 0);
}
