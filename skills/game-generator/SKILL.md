---
name: game-generator
description: Generate an interactive HTML5 educational game to help students learn a specific knowledge point. Use when a student struggles with a concept, is bored, or when gamification would enhance learning. The game should be playable directly in a web browser as a single HTML file.
---

# Game Generator

Generate a self-contained HTML5 game to teach a specific knowledge point.

## When to use
- Student has difficulty understanding a concept after 2+ explanation attempts
- Student seems bored or disengaged
- A knowledge point is well-suited for gamification (math operations, vocabulary, etc.)

## Process
1. Identify the knowledge point and student's current level from KNOWLEDGE.md
2. Read the game template: use read_skill tool with asset_file="assets/game-template.html"
3. Generate a complete, playable HTML5 game based on the template
4. Call render_content with type="game", a descriptive title, and the complete HTML

## Game Design Principles
- Keep it simple: 1-3 levels, clear win condition
- Immediate feedback on correct/incorrect answers
- Show the correct answer after wrong attempt
- Track score and show progress
- Include the knowledge point explanation in the game UI
- Mobile-friendly (touch events)
- Bright, encouraging visual design

## Examples
- Fraction addition: pizza slicing game where player must combine slices
- Vocabulary: word matching card flip game
- Math operations: balloon popping game with correct answers
- History dates: timeline ordering game
