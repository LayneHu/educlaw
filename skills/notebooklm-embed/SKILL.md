---
name: notebooklm-embed
description: Create a NotebookLM study resource for a topic. Use when a student needs to deeply study a topic, wants to create study notes, or needs an interactive document. Generates an embed or link to NotebookLM.
---

# NotebookLM Embed

Create a structured study resource with NotebookLM integration.

## When to use
- Student needs to study a complex topic deeply
- Creating study notes for an exam
- Building a research project
- Long-form content review

## Process
1. Identify the topic from the conversation
2. Generate a structured study guide HTML
3. Include a NotebookLM link/button for deeper research
4. Call render_content with type="embed"

## Output Format
Create an HTML study card with:
- Key concepts summary
- Important terms glossary
- Practice questions
- Link to create a NotebookLM notebook
