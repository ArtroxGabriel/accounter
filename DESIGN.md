# Design System Specification: The Mindful Ledger

## 1. Overview & Creative North Star
**Creative North Star: "The Financial Sanctuary"**

This design system rejects the high-stress, spreadsheet-dense aesthetic of traditional finance apps in favor of a "Financial Sanctuary." The goal is to transform expense tracking from a chore into a moment of calm reflection. We achieve this through **Editorial Softness**—using generous white space, asymmetrical layouts that guide the eye naturally, and a "paper-on-glass" layering logic.

By moving away from rigid grids and 1px borders, we create an interface that feels curated and intentional. The experience should feel like flipping through a high-end architectural magazine: airy, organized, and deeply professional.

---

## 2. Colors & Surface Logic

### The Palette
The color strategy uses a base of sophisticated neutrals to ground the experience, with high-intent accents for financial status.

- **Primary (`#006c5a`):** A deep, scholarly teal used for high-level brand moments and primary actions.
- **Positive Balance (`#9cf3dc`):** A soft mint used in containers to signify growth and safety.
- **Negative Balance (`#fc8585`):** A gentle coral/red used to flag expenses without inducing panic.
- **Background (`#f9f9f8`):** An off-white, "bone" finish that is easier on the eyes than pure white.

### The "No-Line" Rule
**Borders are strictly prohibited for sectioning.** To define boundaries, designers must use tonal shifts. A `surface-container-low` section sitting on a `surface` background provides all the definition a user needs. This keeps the UI "breathable."

### Surface Hierarchy & Nesting
Treat the UI as physical layers of fine paper. Use the following tiers to define importance:
1. **Base Layer:** `surface` (`#f9f9f8`)
2. **Sectioning:** `surface-container-low` (`#f2f4f3`)
3. **Actionable Cards:** `surface-container-lowest` (`#ffffff`) for maximum lift.
4. **In-Page Modals:** `surface-container-highest` (`#dee4e2`) for elements that need to feel "closer" to the user.

### The "Glass & Gradient" Rule
To add soul to the interface, use a **Backdrop Blur (12px - 20px)** on floating navigation bars or top headers using a semi-transparent `surface` color. For primary CTAs, apply a subtle linear gradient from `primary` to `primary_dim` to create a soft, pillowy depth.

---

## 3. Typography
We utilize a dual-typeface system to balance character with utility.

* **Headlines & Titles (Manrope):** A modern geometric sans-serif with an open aperture. It feels approachable yet authoritative.
* *Display-LG (3.5rem):* Use for massive "Total Balance" hero moments.
* *Headline-SM (1.5rem):* Use for category headers.
* **Utility & Labels (Inter):** Chosen for its exceptional legibility at small sizes.
* *Label-MD (0.75rem):* Use for micro-data, timestamps, and secondary metadata.

**Editorial Scaling:** Don't be afraid of extreme scale. A very large `display-sm` balance next to a tiny `label-md` date creates a sophisticated, high-end hierarchy that looks designed, not just "placed."

---

## 4. Elevation & Depth

### The Layering Principle
Depth is achieved through **Tonal Stacking**.
* *Example:* Place a `surface-container-lowest` card (White) inside a `surface-container` (Light Grey) area. The contrast creates a natural "lift" without the clutter of a shadow.

### Ambient Shadows
Shadows should be "felt, not seen."
- **Value:** Use a 24px-32px blur with only 4% opacity.
- **Tint:** The shadow color must be a tinted version of `on-surface` (`#2e3433`), never pure black. This mimics natural ambient light.

### The "Ghost Border" Fallback
If accessibility requires a container boundary, use a **Ghost Border**: `outline-variant` (`#adb3b2`) at **15% opacity**. It should be a mere suggestion of a line.

---

## 5. Components

### Buttons
* **Primary:** High-pill shape (`rounded-full`). Background: `primary`. Text: `on-primary`. No shadow.
* **Secondary:** `surface-container-high` background. Text: `on-secondary-container`. This should feel integrated into the surface.

### Input Fields
* **Style:** Minimalist under-line or subtle background fill (`surface-container-low`).
* **Focus State:** Transition to a `primary` "Ghost Border" (20% opacity). Never use heavy strokes.

### Cards & Transaction Lists
* **Rule:** **No dividers.**
* **Separation:** Use `16px` or `24px` of vertical whitespace to separate transaction days. Within a day, use a subtle background shift on hover to indicate interactivity.
* **Visual Cue:** Use a small 4px vertical "accent pill" of `primary-container` (Mint) or `tertiary-container` (Coral) on the far left of a transaction card to indicate income vs. expense.

### The "Financial Health" Gauge
A custom component for this system. A wide, low-profile progress bar using `surface-container-highest` as the track and a gradient of `primary` to `primary_fixed_dim` as the indicator.

---

## 6. Do's and Don'ts

### Do
* **Do** use asymmetrical margins (e.g., 24px left, 48px right) for editorial layouts in tablet/desktop views.
* **Do** embrace "Empty Space." If a screen only has one piece of data, let it sit center-stage with massive typography.
* **Do** use `rounded-xl` (12px) for most containers to maintain a soft, friendly hand-feel.

### Don't
* **Don't** use 1px solid `#000` or `#DDD` borders. It breaks the "Financial Sanctuary" immersion.
* **Don't** use "Alert Red." Always use the `tertiary` (`#a13f41`) or `error_container` (`#fa746f`) to keep the emotional volume low.
* **Don't** overcrowd the dashboard. If a piece of data isn't essential for a 5-second glance, move it to a sub-page.
