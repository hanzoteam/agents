// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import {IntlProvider} from 'react-intl';

import {getUserMCPTools, refreshUserMCPTools} from '@/client';

import ToolProviderPopover, {UserMCPServerInfo} from './tool_provider_popover';

jest.mock('@/client', () => ({
    disconnectMCPOAuth: jest.fn(),
    getUserMCPTools: jest.fn(),
    refreshUserMCPTools: jest.fn(),
    updateUserToolPreferences: jest.fn(),
}));

jest.mock('@/hooks/use_mcp_connection_events', () => ({
    useMCPConnectionEvents: jest.fn(),
}));

jest.mock('react-intl', () => ({
    FormattedMessage: ({defaultMessage}: {defaultMessage: string}) => defaultMessage,
    IntlProvider: ({children}: {children: React.ReactNode}) => children,
    useIntl: () => ({
        formatMessage: ({defaultMessage}: {defaultMessage: string}) => defaultMessage,
    }),
}));

// Total, not enumerated: this component pulls in the system-console tree, which
// reaches for icons this file never names. Listing only the two it asserts on
// left the rest undefined, and `styled(undefined)` throws at module load — a
// failure that reads as a styled-components bug three files away.
jest.mock('@hanzoteam/compass-icons/components', () => new Proxy({
    ChevronDownIcon: () => <span data-testid='chevron-icon'/>,
    RefreshIcon: () => <span data-testid='refresh-icon'/>,
} as Record<string, () => JSX.Element>, {

    // Named icons keep their test ids; any OTHER *Icon this component's tree
    // reaches for gets a blank stub, because `styled(undefined)` throws at module
    // load and reads as a styled-components bug three files away. Everything that
    // is not an icon falls through untouched — React probes the module for
    // `$$typeof` and friends, and answering those with a component breaks it.
    get: (named, name) => (typeof name === 'string' && name.endsWith('Icon') && !(name in named) ?
        () => <span/> :
        named[name as string]),
}));

const mockGetUserMCPTools = getUserMCPTools as jest.MockedFunction<typeof getUserMCPTools>;
const mockRefreshUserMCPTools = refreshUserMCPTools as jest.MockedFunction<typeof refreshUserMCPTools>;

const initialServer: UserMCPServerInfo = {
    name: 'Initial Server',
    serverOrigin: 'https://initial.example.com',
    authenticated: true,
    needsOAuth: false,
    tools: [],
};

const refreshedServer: UserMCPServerInfo = {
    name: 'Refreshed Server',
    serverOrigin: 'https://refreshed.example.com',
    authenticated: true,
    needsOAuth: false,
    tools: [],
};

function renderComponent() {
    return render(
        <IntlProvider locale='en'>
            <ToolProviderPopover
                disabledServers={[]}
                onDisabledServersChange={jest.fn()}
                preloadedServers={[initialServer]}
                autoEnableNewMCPTools={true}
            />
        </IntlProvider>,
    );
}

describe('ToolProviderPopover', () => {
    beforeEach(() => {
        mockGetUserMCPTools.mockResolvedValue({servers: [initialServer]});
        mockRefreshUserMCPTools.mockResolvedValue({servers: [refreshedServer]});
    });

    afterEach(() => {
        jest.clearAllMocks();
    });

    test('refresh button forces a user tools refresh', async () => {
        renderComponent();

        fireEvent.click(screen.getByRole('button', {name: 'Tools'}));
        fireEvent.click(await screen.findByRole('button', {name: 'Refresh tool providers'}));

        await waitFor(() => expect(mockRefreshUserMCPTools).toHaveBeenCalledTimes(1));
        expect(screen.getByText('Refreshed Server')).not.toBeNull();
    });
});
