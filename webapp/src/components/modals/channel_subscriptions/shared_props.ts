// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import {Theme} from 'mattermost-redux/types/preferences';
import {Channel} from 'mattermost-redux/types/channels';

import {IssueMetadata, ChannelSubscription, Instance, APIResponse, AllProjectMetadata, GetConnectedResponse, ProjectStatuses} from 'types/model';

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
    getProjectStatuses: (instanceID: string, projectID: string) => Promise<APIResponse<ProjectStatuses[]>>;
    fetchChannelSubscriptions: (channelId: string) => Promise<APIResponse<ChannelSubscription[]>>;
    getConnected: () => Promise<GetConnectedResponse>;
    close: () => void;
    sendEphemeralPost: (message: string) => void;
    securityLevelEmptyForJiraSubscriptions: boolean;
};
