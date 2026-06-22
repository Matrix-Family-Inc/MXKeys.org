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

import type { Meta, StoryObj } from '@storybook/react';

import { NotaryLookupForm } from './notary-lookup-form';

const meta: Meta<typeof NotaryLookupForm> = {
  title: 'Features / NotaryLookupForm',
  component: NotaryLookupForm,
  args: {
    // In Storybook we want to observe what the component would send
    // without a backend; console.info mirrors the "Actions" addon for
    // the barebones (no-addons) setup we currently run with.
    onSubmit: (input) => {
      console.info('[storybook] submit', input);
    },
  },
};
export default meta;

type Story = StoryObj<typeof NotaryLookupForm>;

export const Idle: Story = {};

export const Busy: Story = {
  args: { busy: true },
};
