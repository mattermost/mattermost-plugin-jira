import {ProjectMetadata, IssueMetadata, ChannelSubscription} from 'types/model';

export type SharedProps = {
    channel: {id: string; name: string; display_name: string} | null;
    theme: any;
    jiraProjectMetadata: ProjectMetadata;
    jiraIssueMetadata: IssueMetadata | null;
    channelSubscriptions: ChannelSubscription[];
    omitDisplayName: boolean;
    createChannelSubscription: (sub: ChannelSubscription) => Promise<any>;
    deleteChannelSubscription: (sub: ChannelSubscription) => Promise<any>;
    editChannelSubscription: (sub: ChannelSubscription) => Promise<any>;
    fetchJiraIssueMetadataForProjects: (projectKeys: string[]) => Promise<any>;
    clearIssueMetadata: () => void;
    close: () => void;
};
