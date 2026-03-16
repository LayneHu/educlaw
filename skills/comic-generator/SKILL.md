---
name: comic-generator
description: Generate an educational comic strip to explain a concept or tell a learning story. Use when introducing concepts visually, making learning fun, or retelling historical/scientific stories. Creates a 6-panel comic with emoji characters and colorful CSS backgrounds.
---

# Comic Generator

Generate a self-contained HTML comic strip to teach or illustrate a concept.

## When to use
- Student wants to see a concept explained as a story
- Topic involves a narrative (history, biography, scientific discovery)
- Making abstract concepts concrete through visual storytelling
- Student asks for a comic, story, or visual explanation

## Process
1. Identify the story/concept and the key moments (aim for 6 panels)
2. Read the comic template: use read_skill tool with asset_file="assets/comic-template.html"
3. Modify the COMIC_CONFIG object:
   - title: Comic title in Chinese
   - panels: Array of 6 panel objects, each with:
     - bg: background type ("sky", "classroom", "night", "nature", "space", "lab")
     - character: emoji character(s) (e.g., "🧒", "🤖", "👩‍🏫", "🔬", "🌍")
     - speech: dialogue or caption text (keep under 40 characters)
     - caption: small panel label (e.g., "第一幕", "发现！", "结局")
4. Call render_content with type="visual", a descriptive title, and the complete modified HTML

## Story Arc Guidelines
- Panel 1: Set the scene / introduce problem
- Panel 2: Rising action / exploration
- Panel 3: Discovery / key insight
- Panel 4: Explanation / "aha!" moment
- Panel 5: Application / practice
- Panel 6: Resolution / summary / encouragement

## Examples
- Newton's apple: 6 panels showing Newton sitting, apple falling, thinking, calculating, eureka, sharing discovery
- Fraction learning: 6 panels with student struggling, AI explaining with pizza, practicing, game, success, summary
- History event: 6 panels narrating key moments of a historical event
