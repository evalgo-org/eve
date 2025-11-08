# EVE Corporate Identity & Design System

This document defines the official EVE visual identity and design system. All EVE services, documentation, and user interfaces should follow these guidelines for consistency.

## Color Palette

### Primary Colors

```css
--bg-primary: #f5f7fa        /* Light gray background */
--bg-secondary: #ffffff      /* White cards and panels */
--bg-hover: #f7fafc          /* Hover state background */
--text-primary: #2d3748      /* Dark gray text */
--text-secondary: #718096    /* Medium gray text */
--text-muted: #a0aec0        /* Light gray text */
--border-color: #e2e8f0      /* Border and divider color */
```

### Accent Colors

```css
--accent-primary: #4a5568    /* Primary accent (buttons, links) */
--accent-hover: #2d3748      /* Hover state for accents */
```

### Status Colors (Traffic Light System)

```css
--status-operational: #10b981   /* Green - service operational */
--status-degraded: #f59e0b      /* Orange - degraded performance */
--status-outage: #ef4444        /* Red - service outage */
```

### Usage Guidelines

- **Backgrounds**: Use `--bg-primary` for page backgrounds, `--bg-secondary` for cards/panels
- **Text**: Use `--text-primary` for headings and primary content, `--text-secondary` for descriptions
- **Borders**: Always use `--border-color` for consistency
- **Status**: Only use status colors for actual service status (never for branding)
- **Accents**: Use sparingly for CTAs and important UI elements

## Typography

### Font Stack

```css
font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto',
             'Oxygen', 'Ubuntu', 'Cantarell', sans-serif;
```

### Type Scale

```css
/* Headings */
h1: 2.5rem (40px), font-weight: 700, line-height: 1.2
h2: 2rem (32px), font-weight: 700, line-height: 1.3
h3: 1.5rem (24px), font-weight: 600, line-height: 1.4
h4: 1.25rem (20px), font-weight: 600, line-height: 1.5

/* Body */
body: 1rem (16px), font-weight: 400, line-height: 1.6
lead: 1.15rem (18.4px), line-height: 1.7
small: 0.875rem (14px), line-height: 1.5
```

### Typography Guidelines

- Use **bold (700)** for main headings
- Use **semibold (600)** for subheadings
- Use **medium (500)** for labels and metadata
- Use **regular (400)** for body text
- Line height: 1.6 for body text, tighter for headings
- Letter spacing: -0.5px for large headings, normal for body

## Spacing System

```css
--spacing-sm: 0.5rem    /* 8px */
--spacing-md: 1rem      /* 16px */
--spacing-lg: 2rem      /* 32px */
--spacing-xl: 4rem      /* 64px */
```

### Spacing Guidelines

- Use consistent spacing from the scale above
- Component padding: `--spacing-lg` (32px)
- Section margins: `--spacing-xl` (64px)
- Element gaps: `--spacing-md` (16px)
- Tight spacing: `--spacing-sm` (8px)

## Components

### Buttons

```css
/* Primary Button */
background: var(--accent-primary);
color: white;
padding: 0.75rem 1.5rem;
border-radius: 8px;
font-weight: 600;
border: 2px solid transparent;

/* Secondary Button */
background: transparent;
color: var(--accent-primary);
border: 2px solid var(--accent-primary);
```

### Cards

```css
background: var(--bg-secondary);
padding: 2rem;
border-radius: 12px;
box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
```

### Status Badges

```css
padding: 0.5rem 1.25rem;
border-radius: 50px;
font-size: 0.85rem;
font-weight: 700;
text-transform: uppercase;
letter-spacing: 0.5px;

/* Colors based on status */
.operational: background: var(--status-operational);
.degraded: background: var(--status-degraded);
.outage: background: var(--status-outage);
```

### Navigation

```css
background: var(--bg-secondary);
border-bottom: 2px solid var(--border-color);
padding: var(--spacing-md) 0;
box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
```

## Layout

### Container Width

```css
max-width: 1200px;
margin: 0 auto;
padding: 0 1rem;
```

### Grid Patterns

```css
/* Feature Grid */
display: grid;
grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
gap: 2rem;

/* Service List */
grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
gap: 1rem;
```

## Shadows

```css
/* Light shadow (cards) */
box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);

/* Medium shadow (hover) */
box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);

/* Strong shadow (modals) */
box-shadow: 0 4px 16px rgba(0, 0, 0, 0.15);
```

## Border Radius

```css
/* Buttons, inputs */
border-radius: 8px;

/* Cards */
border-radius: 12px;

/* Pills, badges */
border-radius: 50px;
```

## Transitions

```css
/* Standard transition */
transition: all 0.2s ease;

/* Hover effects */
transition: transform 0.2s, box-shadow 0.2s;
```

## Responsive Breakpoints

```css
/* Mobile */
@media (max-width: 768px)

/* Tablet */
@media (min-width: 769px) and (max-width: 1024px)

/* Desktop */
@media (min-width: 1025px)
```

## Design Principles

1. **Neutral First**: Use neutral colors for 90% of the UI, reserve colors for status/actions
2. **High Contrast**: Ensure text is always readable (4.5:1 minimum WCAG AA)
3. **Consistent Spacing**: Always use spacing scale, never arbitrary values
4. **Clean & Minimal**: Remove unnecessary visual elements
5. **Instant Recognition**: Status should be immediately clear (traffic lights)
6. **Accessible**: Follow WCAG 2.1 AA guidelines
7. **Mobile First**: Design for mobile, enhance for desktop

## Anti-Patterns (Do Not Use)

❌ Gradients for branding
❌ Red or orange for brand colors
❌ Multiple competing accent colors
❌ Arbitrary spacing (use scale)
❌ Emoji in production UI (unless user-generated)
❌ Complex shadows or effects
❌ Centered text for long-form content
❌ Small touch targets (<44px)

## Implementation

### For New Services

1. Copy `/docs/assets/css/main.css` as base
2. Import CSS variables at top of file
3. Use semantic class names
4. Test on mobile first
5. Validate color contrast

### For Documentation

1. Use same HTML structure as eve.evalgo.org
2. Include same navigation
3. Use consistent footer
4. Match spacing and typography

### For Status Displays

1. Always use traffic light colors
2. Include uptime percentages
3. Show response times
4. Update in real-time when possible

## Version History

- **v1.0** (2025-01-08): Initial design system based on statuspageservice and eve.evalgo.org

## References

- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [Material Design Color System](https://material.io/design/color)
- [Tailwind Color Palette](https://tailwindcss.com/docs/customizing-colors)

---

**Maintained by**: EVE Platform Team
**Last Updated**: 2025-01-08
**Status**: Official Corporate Identity
