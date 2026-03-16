---
name: visual-explainer
description: Generate a visual explanation using SVG or Canvas animation to illustrate a concept. Use when a student is a visual learner or when a concept is best understood through diagrams, charts, or animations.
---

# Visual Explainer

Generate an interactive visual explanation for a concept.

## When to use
- Student is a visual learner (noted in INTERESTS.md or PROFILE.md)
- Concept involves spatial relationships (geometry, fractions, etc.)
- Abstract concept benefits from visualization
- Student says "I don't understand" after text explanation

## Process
1. Identify the concept to visualize
2. Create an SVG or Canvas-based HTML visualization
3. Call render_content with type="visual"

## Visualization Types
- **Number lines**: for fractions, integers, inequalities
- **Pie/bar charts**: for fractions, percentages, statistics
- **Animated steps**: for algorithms, processes
- **Diagrams**: for science concepts, geometry
- **Timelines**: for history, sequences

## Technical Notes
- Use SVG for static/simple animations
- Use Canvas for complex animations
- Include interactive elements (hover, click)
- Use bright colors and clear labels
- Mobile-friendly dimensions
