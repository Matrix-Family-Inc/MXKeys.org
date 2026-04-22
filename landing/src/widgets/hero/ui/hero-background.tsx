/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Wed Apr 22 2026 UTC
 * Status: Created
 */

/**
 * Decorative diagonal-stripes backdrop for the hero section. The
 * actual pattern is defined once in `src/index.css` as
 * `.bg-diagonal-stripes` so there are no inline JSX styles and the
 * pattern is reusable by other widgets without duplication.
 *
 * Positioned `absolute inset-0` so the parent `<section>` owns
 * layout and the pattern acts purely as a visual wash.
 */
export function HeroBackground() {
  return (
    <div className="absolute inset-0 opacity-[0.03]" aria-hidden="true">
      <div className="absolute inset-0 bg-diagonal-stripes" />
    </div>
  );
}
