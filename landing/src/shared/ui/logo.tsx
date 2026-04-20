/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Updated
 */

type LogoProps = {
  size?: number;
  className?: string;
};

/**
 * Logo renders the MXKeys mark as inline SVG. Kept as a shared primitive
 * so operators forking the landing for their own branded notary can swap
 * this file without touching widgets.
 */
export function Logo({ size = 32, className }: LogoProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 64 64"
      width={size}
      height={size}
      role="img"
      aria-label="MXKeys logo"
      className={className}
    >
      <defs>
        <linearGradient id="mxkeys-gradient" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="#3D9970" />
          <stop offset="100%" stopColor="#2E7D58" />
        </linearGradient>
      </defs>
      <rect width="64" height="64" rx="12" fill="url(#mxkeys-gradient)" />
      <path
        d="M20 18 L32 32 L20 46 M32 32 L44 18 M32 32 L44 46"
        stroke="white"
        strokeWidth="4"
        strokeLinecap="round"
        strokeLinejoin="round"
        fill="none"
      />
    </svg>
  );
}
