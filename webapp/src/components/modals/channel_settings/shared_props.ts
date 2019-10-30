import {ProjectMetadata, IssueMetadata, ChannelSubscription, Channel} from 'types/model';

export type SharedProps = {
    channel: Channel;
    theme: any;
    jiraProjectMetadata: ProjectMetadata;
    jiraIssueMetadata: IssueMetadata | null;
    channelSubscriptions: ChannelSubscription[];
    createChannelSubscription: (sub: ChannelSubscription) => Promise<any>;
    deleteChannelSubscription: (sub: ChannelSubscription) => Promise<any>;
    editChannelSubscription: (sub: ChannelSubscription) => Promise<any>;
    fetchJiraIssueMetadataForProjects: (projectKeys: string[]) => Promise<any>;
    clearIssueMetadata: () => void;
};
