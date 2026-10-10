import { Client, GatewayIntentBits, SlashCommandBuilder, REST, Routes, ChatInputCommandInteraction } from "discord.js";
import { config, QUESTS } from "../config.js";
import { verifyTxOwnership } from "../arborClient.js";
import { getIdentityByDiscordId, findFeedbackByTx, insertFeedback } from "../db/store.js";

/**
 * Feedback bonus XP is manual and reviewed by a human — this bot only
 * verifies and LOGS submissions as pending_review. It never submits an
 * XP credit itself. See design doc section 6 ("Feedback bonus XP —
 * manual review"). This also means the bot needs no chain-write key,
 * no privileged signer — it's a read-only verifier + a database writer.
 */

const feedbackCommand = new SlashCommandBuilder()
  .setName("feedback")
  .setDescription("Submit feedback for a completed Arbor quest")
  .addStringOption((opt) => opt.setName("quest").setDescription("Quest ID").setRequired(true))
  .addStringOption((opt) => opt.setName("tx").setDescription("Transaction hash for your quest action").setRequired(true))
  .addStringOption((opt) => opt.setName("comments").setDescription("Your feedback").setRequired(true));

async function registerCommands() {
  const rest = new REST({ version: "10" }).setToken(config.discordToken);
  await rest.put(Routes.applicationGuildCommands(client.user!.id, config.discordGuildId), {
    body: [feedbackCommand.toJSON()],
  });
  console.log("[bot] slash commands registered");
}

function lookupAddressForDiscordId(discordId: string): string | undefined {
  return getIdentityByDiscordId(discordId)?.address;
}

async function handleFeedback(interaction: ChatInputCommandInteraction) {
  const questId = interaction.options.getString("quest", true);
  const txHash = interaction.options.getString("tx", true);
  const comments = interaction.options.getString("comments", true);
  const discordId = interaction.user.id;

  const quest = QUESTS.find((q) => q.id === questId);
  if (!quest) {
    return interaction.reply({ content: `Unknown quest id "${questId}". Check #quest-list for valid IDs.`, ephemeral: true });
  }

  const address = lookupAddressForDiscordId(discordId);
  if (!address) {
    return interaction.reply({
      content: "Link your wallet first at arbor.app/link before submitting feedback.",
      ephemeral: true,
    });
  }

  const owns = await verifyTxOwnership(txHash, address);
  if (!owns) {
    return interaction.reply({
      content: "Couldn't verify that transaction — make sure the hash is correct and belongs to your linked wallet.",
      ephemeral: true,
    });
  }

  const existing = findFeedbackByTx(txHash);
  if (existing) {
    return interaction.reply({ content: "Feedback already submitted for this transaction.", ephemeral: true });
  }

  insertFeedback({ address, questId, txHash, comments, submittedAt: Date.now() });

  return interaction.reply({
    content: "✅ Feedback received — thanks! Bonus XP will be reviewed and credited soon. Your quest XP for this action was already credited automatically.",
    ephemeral: true,
  });
}

const client = new Client({ intents: [GatewayIntentBits.Guilds] });

client.once("ready", async () => {
  console.log(`[bot] logged in as ${client.user?.tag}`);
  await registerCommands();
});

client.on("interactionCreate", async (interaction) => {
  if (!interaction.isChatInputCommand()) return;
  if (interaction.commandName === "feedback") {
    try {
      await handleFeedback(interaction);
    } catch (err) {
      console.error("[bot] feedback handler error:", err);
      if (!interaction.replied) {
        await interaction.reply({ content: "Something went wrong, try again in a moment.", ephemeral: true });
      }
    }
  }
});

client.login(config.discordToken);
