# PDF ticker rename checklist — CLIQ → CPLQ

The two PDFs are binary with no editable source in the repo, so they could **not** be edited
automatically. Apply the changes below in whatever tool produced them (Google Docs / export
pipeline), then re-export.

## Global rule

Replace **every standalone occurrence of the ticker `CLIQ` with `CPLQ`** (including in numbers
like `100,000,000 CLIQ` → `100,000,000 CPLQ` and allocations like `22M CLIQ` → `22M CPLQ`).

**Do NOT change** — these are not the ticker:
- `canoLiq` — the product/protocol name (appears in titles, headers, footers, body).
- `CNPY` and `cCNPY` — the Canopy native token and its liquid-staking receipt.

A blind find-and-replace of the exact string `CLIQ` → `CPLQ` is safe: `canoLiq`, `CNPY`, and
`cCNPY` do not contain the substring `CLIQ`.

---

## canoLiq_Tokenomics_v1.2.pdf (9 pages)

- **Running header (every page, top-right):** "CLIQ Token Design v1.2" → "CPLQ Token Design v1.2"
- **p1 (cover):** subtitle "CLIQ Token Design & Protocol Economics"; the big figure "100,000,000 CLIQ"
- **p2 §1 Token Overview:** opening "CLIQ is the governance and value-capture token…"; "emission complexity: CLIQ is not a high-emission…"; property table **Token Name = CLIQ**; "Secondary Token … distinct from CLIQ"
- **p3:** §2.1 heading "22M CLIQ" + "their CLIQ vesting should not unlock"; §2.2 "20M CLIQ"; §2.3 "15M CLIQ", "Governed entirely by CLIQ holders", "> 1M CLIQ equivalent"; §2.4 "15M CLIQ", "CLIQ pairs", "CLIQ emissions activate only as needed", "reduce actual CLIQ emission"
- **p4:** §2.5 "12M CLIQ"; §2.6 "10M CLIQ", "500K CLIQ per partner"; §2.7 "6M CLIQ"; §3 "value accrual to CLIQ holders"
- **p5:** fee table row **"CLIQ Buyback"** + "Open-market CLIQ purchase; … locked CLIQ stakers"; §3.3 "CLIQ holders may vote…"; §4 heading "CLIQ Value Accrual Mechanics"; §4.1 "purchase CLIQ on the open market", "purchased CLIQ is to burn", "CLIQ holders may vote to redirect buyback proceeds to locked CLIQ stakers"; table column "Annual CLIQ Buyback (15%)"
- **p6:** §4.2 "CLIQ stakers who time-lock"; §5 "why CNPY subsidies matter for CLIQ tokenomics"; §5.1 "own token (CLIQ)", "freshly minted CLIQ", "15M CLIQ liquidity allocation"; callout "Why this matters for CLIQ holders" + "Every CLIQ that is NOT emitted in months 1–6 is CLIQ that is not sold…" + "preserves CLIQ value"
- **p7:** §5.2 "zero additional CLIQ required"; §5.3 table "Effective CLIQ Emission", "Impact on CLIQ Supply", "15M CLIQ (full allocation, front-loaded)", "~8–10M CLIQ", "early CLIQ emission", "~5–8M CLIQ"
- **p8:** §7 "CLIQ is the sole governance token…", "require CLIQ holder votes", governance table "> 1M CLIQ equiv."; §8 "any CLIQ holder or CNPY depositor", "CLIQ holders bear this risk", "CLIQ liquidity risk: early CLIQ markets…", "accumulates > 33% of circulating CLIQ"
- **p9 §9 Disclaimer:** "CLIQ and cCNPY tokens involve significant risk" → "CPLQ and cCNPY…"

## canoLiq_Whitepaper_v1.2.pdf (12 pages)

- **p1 Abstract:** "A secondary governance and value-capture token (CLIQ)"
- **p2:** Key Value Propositions → "Governance: CLIQ holders direct protocol parameters…"; rewards-flow table step 5 "15% to CLIQ buyback"
- **p3:** §1.2 "ejected from the validator set by CLIQ governance vote"; footnote "Validator set expansion is governed by CLIQ holders"
- **p6:** §4.2 fee table row **"CLIQ Buyback & Burn"** + "Open-market CLIQ purchases; … locked CLIQ stakers"; §5 heading "CLIQ Tokenomics"; §5.1 "CLIQ is the governance and value-capture token… 100,000,000 CLIQ. CLIQ is not a reward emission token"
- **p7:** §5.3 "CLIQ pairs", "time-locked multisig controlled by CLIQ governance"; §5.4 heading "CLIQ Value Accrual" + "CLIQ accrues value through three mechanisms", "purchase CLIQ on the open market", "burn purchased CLIQ (deflationary)", "CLIQ holders may vote to redirect buyback proceeds to locked CLIQ stakers", "CLIQ stakers with locked positions", "share of buyback CLIQ", "CLIQ holders govern the most critical lever", "making CLIQ a claim on future protocol cash flows"
- **p8:** §6.1 "native token (CLIQ)", "freshly minted CLIQ", "instead of CLIQ", "less inflation pressure on CLIQ"; Concrete Example "emit 500,000 CLIQ", "zero additional CLIQ", "15M CLIQ liquidity allocation", "long-term CLIQ holders"
- **p9:** §6.3 table "CLIQ Emitted (Liquidity Bucket)", "15M CLIQ over 24 months", "Heavy early sell pressure on CLIQ", "~8–10M CLIQ", "early CLIQ emission", "~5–8M CLIQ"; §7 "CLIQ holders vote on restaking allocation policy"; §8.1 heading "Scope of CLIQ Governance" + "CLIQ holders govern all critical protocol parameters"
- **p10:** §8.2 "CLIQ can be staked and time-locked for boosted voting power"
- **p12:** Roadmap "CLIQ vesting contracts; testnet deployment", "CLIQ distribution begins"; §12 Disclaimer "CLIQ and cCNPY tokens involve significant risk"

> Tip: also update the PDF **file names** if you regenerate them — the ticker isn't in the
> filenames today (`canoLiq_*`), so no rename is needed there.
