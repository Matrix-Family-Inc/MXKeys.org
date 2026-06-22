/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
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
