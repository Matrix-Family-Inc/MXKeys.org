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

import { create } from 'zustand';

type MobileNavState = {
  open: boolean;
  toggle: () => void;
  close: () => void;
};

/**
 * useMobileNav centralizes the mobile navigation open/closed state so the
 * header toggle and any nested link can cooperate without prop drilling or
 * context wiring. Kept deliberately tiny: a single boolean + two mutators.
 */
export const useMobileNav = create<MobileNavState>((set) => ({
  open: false,
  toggle: () => set((state) => ({ open: !state.open })),
  close: () => set({ open: false }),
}));
