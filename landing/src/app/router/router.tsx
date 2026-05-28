/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { createRootRoute, createRoute, createRouter, Outlet } from '@tanstack/react-router';

import { HomePage } from '../../pages/home';

/**
 * rootRoute holds the shell (currently just <Outlet />; global chrome lives
 * inside HomePage so a future /docs, /status, /playground can compose their
 * own header/footer if they diverge from the marketing pages).
 */
const rootRoute = createRootRoute({
  component: () => <Outlet />,
});

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: HomePage,
});

const routeTree = rootRoute.addChildren([indexRoute]);

/**
 * router is the singleton used by the RouterProvider in app/providers.
 * Defining routes in TanStack Router (rather than hand-rolled conditional
 * rendering) costs little now and pays off as soon as the first real
 * secondary page ships.
 */
export const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
