import {ProjectMetadata, IssueMetadata, ChannelSubscription} from 'types/model';

export type SharedProps = {
    channel: {id: string; display_name: string};
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
