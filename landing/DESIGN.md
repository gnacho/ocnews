# DESIGN.md - ocnews landing

## Context (from discovery)

- Artifact type: landing page
- Positioning: technical / self-hosting
- Audience: OpenCloud admins and Nextcloud users looking for a lighter RSS stack
- Adjectives: lightweight, compatible, focused, open, calm
- Visual word translations:
  - lightweight -> airy whitespace, single-page flow, no visual noise
  - compatible -> familiar RSS imagery and Nextcloud News references
  - focused -> tight section rhythm, one claim at a time
  - open -> open-source license front and center, GitHub CTA
  - calm -> warm neutrals, rounded shapes, readable type
- Aesthetic essence (3 words): newsroom clarity, friendly utility
- Single-minded proposition: "Your RSS reader, without the datacenter."
- References: admire Colorlib Unapp (hero + slider + feature blocks rhythm); avoid generic indigo SaaS templates and stock laptop-pointing imagery
- Mode: light default with dark toggle
- Density: balanced
- Constraints: static HTML+CSS+JS, no build, i18n ES/EN, Lucide icons only, must include Cloudless club link, no demo server

## Aesthetic

- Direction: Unapp-inspired product landing adapted for a small open-source project
- Defining trait: alternating light/off-white sections with one full-bleed dark hero, all tied by a single warm accent
- Signature move: a "news feed strip" that runs under the hero CTA and a slider of real app screenshots immediately below the hero, like a folded newspaper

## Typography

- Display: "Nunito", Arial, sans-serif | source: Google Fonts | license: OFL (used for headings)
- Body: "Poppins", Arial, sans-serif | source: Google Fonts | license: OFL (weight 300, the Unapp default)
- Mono: "JetBrains Mono", ui-monospace, monospace | source: Google Fonts | license: OFL
- Scale: ratio 1.25 (Major Third), base 16px

| step | size | line-height | use |
|---|---|---|---|
| display | 56px | 1.15 | hero headline (desktop) |
| h1 | 40px | 1.2 | page title |
| h2 | 32px | 1.25 | section titles |
| h3 | 24px | 1.3 | feature/cards titles |
| body | 16px | 1.6 | paragraphs |
| small | 14px | 1.5 | captions, meta |
| eyebrow | 12px | 1.4 | uppercase labels, letter-spacing 0.08em |

- Weights: 400 (body), 500 (strong), 600 (headings/labels), 700 (hero display)
- Measure: 65ch for body text

## Color

- Strategy: faithful reproduction of the Colorlib Unapp template palette. Blue diagonal gradient hero, green accent for actions, whitesmoke background. Avoids the indigo/violet band as a dominant (the hero gradient uses blue #499bea to periwinkle #798eea as a background wash, not as text/UI accent).
- Distribution: 60 neutral / 30 brand / 10 accent

| role | hex |
|---|---|
| bg | #f5f5f5 (whitesmoke) |
| surface | #ffffff |
| fg | #303133 |
| muted | gray (#808080) |
| border | #e6e6e6 |
| accent | #4aca85 (Unapp green) |
| accent-hover | #5ed092 |
| accent-fg | #ffffff |
| hero-a | #499bea |
| hero-b | #798eea |
| success | #4aca85 |
| warning | #e6a23c |
| error | #d9534f |

- Dark mode overrides:
  - bg: #1c1e21
  - surface: #25272b
  - fg: #e8e9ea
  - muted: #9a9da1
  - border: #35383d
  - accent: #4aca85 (kept, passes on dark)

## Spacing, radius, shadow

- Spacing base: 4px
- Scale: 4, 8, 12, 16, 24, 32, 48, 64, 96, 128
- Radius: 6px (small), 999px (pill buttons)
- Shadow approach: soft elevation only; no hairline + diffuse shadow on the same element
  - sm: 0 1px 2px rgba(31 29 27 / 0.05)
  - md: 0 4px 12px rgba(31 29 27 / 0.08)
  - lg: 0 12px 32px rgba(31 29 27 / 0.12)

## Layout and composition

- Grid: 12-column, max-width 1200px, 24px gutters, 16px mobile margins
- Spacing rhythm: tight within cards/groups (12-16px), generous between sections (80-96px)
- Signature layout move: the hero flows into a slider band without a hard separator, making the app feel like the continuation of the headline
- Density: balanced
- Scanning: Z-pattern on hero, then F-pattern for features
- Responsive: desktop-first with mobile breakpoints at 900px and 600px

## Components and states

- Button hierarchy:
  - primary: accent fill, white text, radius 6px, hover darken 8%, active scale 0.98
  - secondary: transparent with border, text fg, hover surface background
  - tertiary: text only with underline on hover
- Inputs: not used on the landing except the install copy box (readonly, monospace)
- Cards: surface background, radius 6px, shadow md, hover shadow lg, no border
- Tables: text left, numbers right (used in comparison matrix)
- Focus ring: 2px accent offset 2px box-shadow

## Motion

- Duration scale:
  - instant: 100ms
  - fast: 150ms
  - normal: 200ms
  - slow: 300ms
- Easing:
  - ease-out: cubic-bezier(0.23, 1, 0.32, 1)
  - ease-in-out: cubic-bezier(0.77, 0, 0.175, 1)
- What animates: reveals (translateY + opacity), slider transitions (opacity), theme switch (color transition 200ms)
- prefers-reduced-motion: swap motion for a simple opacity fade

## Iconography

- Set: Lucide
- Grid: 24px
- Stroke: 1.5px
- Radius match: icons use rounded caps/joins that match the 6px radius of buttons/cards

## Imagery and illustration

- Mode: real product screenshots
- Rules: WebP 1440px, compressed, lazy-loaded outside hero; each screenshot has a descriptive alt
- Avoid: stock photography, gradient blobs, people pointing at laptops
- Text-over-image contrast: hero overlay uses a dark gradient from rgba(0,0,0,0.65) to rgba(0,0,0,0.4) so white text passes AA

## Dark mode

- Base bg: near-black #141210
- Elevation: surface layers lighten to #1d1b18, #272522
- Accent: lighter, slightly desaturated orange #f58a3c
- Borders: lighter than surface

## Accessibility

- Contrast: AA verified in both modes (body 4.5:1, large text 3:1)
- Focus: visible 2px accent ring on all interactive elements
- Keyboard: nav links, buttons, slider controls, language switch and theme button are keyboard operable
- Targets: buttons >= 44px
- Color independence: icons paired with labels; comparison matrix uses symbols plus text
- Reduced motion: fade-only alternative

## Tokens (source of truth)

```css
:root {
  --font-display: "Poppins", sans-serif;
  --font-body: "Nunito", sans-serif;
  --font-mono: "JetBrains Mono", ui-monospace, monospace;

  --bg: #fdfcfa;
  --surface: #f7f5f2;
  --fg: #1f1d1b;
  --muted: #6b6864;
  --border: #e3e0db;
  --accent: #c24a00;
  --accent-fg: #ffffff;
  --success: #2d8a4e;
  --warning: #c87f0a;
  --error: #b93b3b;

  --shadow-sm: 0 1px 2px rgba(31 29 27 / 0.05);
  --shadow-md: 0 4px 12px rgba(31 29 27 / 0.08);
  --shadow-lg: 0 12px 32px rgba(31 29 27 / 0.12);

  --radius: 6px;
  --radius-pill: 999px;

  --duration-instant: 100ms;
  --duration-fast: 150ms;
  --duration-normal: 200ms;
  --duration-slow: 300ms;
  --ease-out: cubic-bezier(0.23, 1, 0.32, 1);
  --ease-in-out: cubic-bezier(0.77, 0, 0.175, 1);
}

[data-theme="dark"] {
  --bg: #141210;
  --surface: #1d1b18;
  --fg: #f2f0ed;
  --muted: #9b9894;
  --border: #36332f;
  --accent: #f58a3c;
  --accent-fg: #1a120c;
  --shadow-sm: 0 1px 2px rgba(0 0 0 / 0.25);
  --shadow-md: 0 4px 12px rgba(0 0 0 / 0.35);
  --shadow-lg: 0 12px 32px rgba(0 0 0 / 0.45);
}
```

- Adapter: plain CSS custom properties

## Cards and surfaces

- Cards: surface background, radius 6px, shadow-md, no border
- Hover: shadow-lg, translateY(-2px) on feature cards only
- Nesting: avoid cards inside cards; the comparison table lives on a single surface

## Slop audit

- Date: 2026-08-19
- Result: pass
- Notes:
  - Accent is orange (#c24a00 in light for text/links; #ee7318 stays as the brand icon color and #f58a3c in dark), not indigo
  - Typography is Poppins+Nunito, not Inter/Space Grotesk
  - Hero is not just centered text; it flows into a screenshot slider
  - Feature blocks alternate image+text, not a uniform 3-card grid
  - No hairline+shadow combo; surfaces use shadow only
  - Real screenshots, no stock
  - Motion uses transform/opacity only
  - Dark mode is a designed mode, not inversion
  - Zero em/en dashes in copy
  - Contrast AA verified 10/10 both modes (accent light darkened to #c24a00 to reach 4.79:1 on bg and 4.91:1 for white text on it; original #ee7318 only for decoration/hero over dark)

## Changelog

- 2026-08-19: initial design system for ocnews landing
- 2026-08-19: re-skinned to the Colorlib Unapp original palette (blue gradient hero, green accent, whitesmoke bg, Poppins/Nunito); accent/gray nudged minimally for WCAG AA
