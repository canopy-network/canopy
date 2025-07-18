#!/usr/bin/env bash

echo "Canopy Order Book:"

canopy query orders | jq -r '
  def green: "\u001b[32m";
  def reset: "\u001b[0m";
  .[] |
  .orders[] |
  "ID: " + green + "\(.id) " + reset +
  "CNPY: \(.amountForSale) " +
  "\tUSDC: \(.requestedAmount) Deadline: \(.buyerChainDeadline)\n" +
  "\tBuyer  USDC: \(.buyerSendAddress) CNPY: \(.buyerReceiveAddress)\n" +
  "\tSeller USDC: \(.sellerReceiveAddress) CNPY: \(.sellersSendAddress)\n" +
  "\tData: \(.data) Committee: \(.committee)\n"
'
echo
