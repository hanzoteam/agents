// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';
import {render, screen} from '@testing-library/react';

// Minimal react-intl shim: ts-jest bypasses babel, so FormattedMessage needs an id at runtime.
jest.mock('react-intl', () => {
    const React = require('react'); // eslint-disable-line @typescript-eslint/no-shadow, no-shadow, global-require

    return {
        __esModule: true,
        IntlProvider: ({children}: {children: React.ReactNode}) => React.createElement(React.Fragment, null, children),
        FormattedMessage: ({defaultMessage}: {defaultMessage?: string}) =>
            React.createElement(React.Fragment, null, defaultMessage ?? ''),
        useIntl: () => ({
            formatMessage: ({defaultMessage}: {defaultMessage?: string}) => defaultMessage ?? '',
        }),
    };
});

// OverlayTrigger renders the overlay alongside children so tests can assert the tooltip text.
jest.mock('react-bootstrap', () => ({
    OverlayTrigger: ({children, overlay}: {children: React.ReactNode; overlay: React.ReactNode}) => <>{children}{overlay}</>,
    Tooltip: ({children}: {children: React.ReactNode}) => <div>{children}</div>,
}), {virtual: true});

// The component reads SiteURL via useSelector; null falls back to window.location.origin.
jest.mock('react-redux', () => ({
    useSelector: jest.fn(() => null),
}));

jest.mock('../../client', () => ({
    __esModule: true,
    getMCPTools: jest.fn().mockResolvedValue({servers: []}),
    clearMCPToolsCache: jest.fn(),
    getVettedToolSeed: jest.fn().mockResolvedValue([]),
    updatePluginServer: jest.fn().mockResolvedValue({}),
}));

/* eslint-disable import/first, import/order */
import {IntlProvider} from 'react-intl';

import MCPServers, {MCPConfig, MCPServerConfig} from './mcp_servers';
/* eslint-enable import/first, import/order */

function makeMCPConfig(servers: MCPServerConfig[] = []): MCPConfig {
    return {
        enabled: true,
        enablePluginServer: false,
        servers,
        embeddedServer: {enabled: true, tool_configs: []},
    };
}

function makeRemoteServer(): MCPServerConfig {
    return {
        name: 'Jira',
        enabled: true,
        baseURL: 'https://mcp.example.com',
        headers: {},
    };
}

function renderServers(mcpConfig: MCPConfig) {
    const onChange = jest.fn();
    return {
        ...render(
            <IntlProvider locale='en'>
                <MCPServers
                    mcpConfig={mcpConfig}
                    onChange={onChange}
                />
            </IntlProvider>,
        ),
        onChange,
    };
}

describe('MCPServers', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    test('remote server UI is shown', async () => {
        renderServers(makeMCPConfig([makeRemoteServer()]));

        const addButton = screen.getByRole('button', {name: /Add Remote MCP Server/});
        expect((addButton as HTMLButtonElement).disabled).toBe(false);
        expect(screen.getByText('Jira')).not.toBeNull();
        expect(screen.getByText('MCP OAuth Callback URL')).not.toBeNull();
    });
});
