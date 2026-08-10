/**
 * Simple CLI for the manual review step. Run with:
 *   tsx src/reviewFeedback.ts list
 *   tsx src/reviewFeedback.ts approve <feedback-id> <bonus-xp>
 *   tsx src/reviewFeedback.ts reject <feedback-id>
 *
 * This is the one deliberately manual gate in the whole system — feedback
 * quality is subjective, so it stays a human decision. Approving here
 * credits XP under a synthetic quest_id of "feedback:<original-quest-id>"
 * so bonus XP is tracked separately from automatic quest-completion XP
 * but still rolls into the same weekly leaderboard totals.
 */
import { weekIdForHeight, dayIdForHeight } from "./config.js";
import { fetchCurrentHeight } from "./arborClient.js";
import { listPendingFeedback, getFeedbackById, reviewFeedback, creditXp } from "./db/store.js";

const [, , cmd, arg1, arg2] = process.argv;

function list() {
  const rows = listPendingFeedback();
  console.table(rows);
}

async function approve(feedbackIdStr: string, bonusXpStr: string) {
  const feedbackId = Number(feedbackIdStr);
  const fb = getFeedbackById(feedbackId);
  if (!fb) return console.error("feedback not found");
  if (fb.status !== "pending_review") return console.error(`already ${fb.status}`);

  const xp = Number(bonusXpStr);
  const height = await fetchCurrentHeight();
  const weekId = weekIdForHeight(height);
  const dayId = dayIdForHeight(height);

  creditXp({
    address: fb.address,
    weekId,
    dayId,
    questId: `feedback:${fb.questId}`,
    txHash: fb.txHash,
    xp,
    creditedAt: Date.now(),
  });

  reviewFeedback(feedbackId, "approved", xp);
  console.log(`approved feedback #${feedbackId}: +${xp} bonus XP to ${fb.address}`);
}

function reject(feedbackIdStr: string) {
  reviewFeedback(Number(feedbackIdStr), "rejected", null);
  console.log(`rejected feedback #${feedbackIdStr}`);
}

switch (cmd) {
  case "list":
    list();
    break;
  case "approve":
    approve(arg1, arg2);
    break;
  case "reject":
    reject(arg1);
    break;
  default:
    console.log("usage: tsx src/reviewFeedback.ts [list|approve <id> <xp>|reject <id>]");
}
