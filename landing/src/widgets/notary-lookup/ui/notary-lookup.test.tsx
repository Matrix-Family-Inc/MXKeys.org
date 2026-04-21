/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { afterEach, describe, expect, it } from 'vitest';
import { cleanup, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

import { NotaryLookupSection } from './notary-lookup';

afterEach(cleanup);

function renderWithQuery(ui: React.ReactElement) {
  const qc = new QueryClient({
    defaultOptions: {
      mutations: { retry: false },
      queries: { retry: false },
    },
  });
  return render(<QueryClientProvider client={qc}>{ui}</QueryClientProvider>);
}

describe('NotaryLookupSection', () => {
  it('queries the notary and renders the returned keys via MSW', async () => {
    renderWithQuery(<NotaryLookupSection />);

    // role=textbox disambiguates the input from section labels that
    // also contain "Matrix server" text.
    await userEvent.type(
      screen.getByRole('textbox', { name: /matrix server/i }),
      'matrix.example.org',
    );
    await userEvent.click(screen.getByRole('button', { name: /check notary keys/i }));

    await waitFor(() => {
      expect(screen.getByText(/matrix\.example\.org/)).toBeInTheDocument();
    });
    expect(screen.getByText(/ed25519:auto/i)).toBeInTheDocument();
  });
});
