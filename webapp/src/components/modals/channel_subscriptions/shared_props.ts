// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Theme} from 'mattermost-redux/selectors/entities/preferences';
import {Channel} from '@mattermost/types/channels';

import {
    APIResponse,
    AllProjectMetadata,
    ChannelSubscription,
    GetConnectedResponse,
    Instance,
    IssueMetadata,
} from 'types/model';

export type SharedProps = {
    channel: Channel | null;
    theme: Theme;
    channelSubscriptions: ChannelSubscription[];
    omitDisplayName: boolean;
    installedInstances: Instance[];
    connectedInstances: Instance[];
    createChannelSubscription: (sub: ChannelSubscription) => Promise<APIResponse<{}>>;
    deleteChannelSubscription: (sub: ChannelSubscription) => Promise<APIResponse<{}>>;
    editChannelSubscription: (sub: ChannelSubscription) => Promise<APIResponse<{}>>;
    fetchJiraProjectMetadataForAllInstances: () => Promise<APIResponse<AllProjectMetadata>>;
    fetchJiraIssueMetadataForProjects: (projectKeys: string[], instanceID: string) => Promise<APIResponse<IssueMetadata>>;
    fetchChannelSubscriptions: (channelId: string) => Promise<APIResponse<ChannelSubscription[]>>;
    getConnected: () => Promise<GetConnectedResponse>;
    close: () => void;
    sendEphemeralPost: (message: string) => void;
    securityLevelEmptyForJiraSubscriptions?: boolean;
};
