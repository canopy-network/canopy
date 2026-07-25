package com.canopy.tutorial

/**
 * Canopy Kotlin Plugin Tutorial
 * 
 * This project demonstrates how to test custom transaction types
 * (faucet and reward) for the Canopy Kotlin plugin.
 * 
 * See TUTORIAL.md in the parent directory for the full tutorial.
 * 
 * To run the integration tests (transactions + custom RPC endpoints):
 *   ./gradlew test
 * 
 * Or use the Makefile:
 *   make test
 */
fun main() {
    println("Canopy Kotlin Plugin Tutorial")
    println("==============================")
    println()
    println("This project contains tests for custom transaction types.")
    println()
    println("To run the integration tests (transactions + custom RPC endpoints):")
    println("  ./gradlew test")
    println()
    println("Or use the Makefile:")
    println("  make test")
    println()
    println("Prerequisites:")
    println("  1. Canopy node must be running with the Kotlin plugin enabled")
    println("  2. The plugin must have faucet and reward transaction types registered")
}
