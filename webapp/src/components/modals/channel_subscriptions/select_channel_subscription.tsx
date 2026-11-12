// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React from 'react';

import {PlusIcon} from '@mattermost/compass-icons/components';
import {Button} from '@mattermost/compass-ui/components/button';
import {Icon} from '@mattermost/compass-ui/components/icon';
import {SectionNotice} from '@mattermost/compass-ui/components/section-notice';
import {Tabs} from '@mattermost/compass-ui/components/tabs';

import {AllProjectMetadata, ChannelSubscription} from 'types/model';

import ConfirmModal from 'components/confirm_modal';
import Modal from 'components/modal';

import {SharedProps} from './shared_props';

export type SubscriptionsTab = 'subscriptions' | 'templates';

const TAB_PANEL_ID = 'jira-subscriptions-panel';

type Props = SharedProps & {
    open?: boolean;
    activeTab: SubscriptionsTab;
    onTabChange: (tab: SubscriptionsTab) => void;
    showEditChannelSubscription: (subscription: ChannelSubscription) => void;
    showEditSubscriptionTemplate: (subscription: ChannelSubscription) => void;
    showCreateChannelSubscription: () => void;
    showCreateSubscriptionTemplate: () => void;
    allProjectMetadata: AllProjectMetadata | null;
};

type State = {
    error: string | null;
    showConfirmModal: boolean;
    subscriptionToDelete: ChannelSubscription | null;
    isTemplate: boolean;
}

export default class SelectChannelSubscriptionInternal extends React.PureComponent<Props, State> {
    state = {
        error: null,
        showConfirmModal: false,
        subscriptionToDelete: null,
        isTemplate: false,
    };

    handleCancelDelete = (): void => {
        this.setState({showConfirmModal: false});
    };

    handleConfirmDelete = (): void => {
        this.setState({showConfirmModal: false});
        if (this.state.isTemplate) {
            this.deleteSubscriptionTemplate(this.state.subscriptionToDelete);
        } else {
            this.deleteChannelSubscription(this.state.subscriptionToDelete);
        }
    };

    handleDeleteChannelSubscription = (sub: ChannelSubscription, isTemplate = false): void => {
        this.setState({
            showConfirmModal: true,
            subscriptionToDelete: sub,
            isTemplate,
        });
    };

    deleteChannelSubscription = (sub: ChannelSubscription): void => {
        this.props.deleteChannelSubscription(sub).then((res: { error?: { message: string } }) => {
            if (res.error) {
                this.setState({error: res.error.message});
            }
        });
    };

    deleteSubscriptionTemplate = (sub: ChannelSubscription): void => {
        this.props.deleteSubscriptionTemplate(sub).then((res: {error?: {message: string}}) => {
            if (res.error) {
                this.setState({error: res.error.message});
            }
        });
    };

    getProjectName = (sub: ChannelSubscription): string => {
        const projectKey = sub.filters.projects[0];
        if (!this.props.allProjectMetadata) {
            return projectKey;
        }

        const instanceData = this.props.allProjectMetadata.find((m) => m.instance_id === sub.instance_id);
        if (instanceData) {
            const project = instanceData.metadata.projects.find((p) => p.value === projectKey);
            if (project) {
                return project.label;
            }
        }

        return projectKey;
    };

    renderRow = (sub: ChannelSubscription, forTemplates = false): JSX.Element => {
        const showInstanceColumn = this.props.installedInstances.length > 1;
        const alias = this.props.installedInstances.filter((instance) => instance.instance_id === sub.instance_id)[0].alias;
        const instanceName = alias || sub.instance_id;
        const onEdit = forTemplates ? this.props.showEditSubscriptionTemplate : this.props.showEditChannelSubscription;

        return (
            <tr
                key={sub.id}
                className='select-channel-subscriptions-row'
            >
                <td>{sub.name || '(no name)'}</td>
                <td>{this.getProjectName(sub)}</td>
                {showInstanceColumn && <td>{instanceName}</td>}
                <td className='jira-subscriptions__actions'>
                    <Button
                        emphasis='link'
                        size='small'
                        onClick={() => onEdit(sub)}
                    >
                        {'Edit'}
                    </Button>
                    <Button
                        emphasis='link'
                        size='small'
                        destructive={true}
                        onClick={() => this.handleDeleteChannelSubscription(sub, forTemplates)}
                    >
                        {'Delete'}
                    </Button>
                </td>
            </tr>
        );
    };

    render(): React.ReactElement {
        const {channel, channelSubscriptions, subscriptionTemplates, omitDisplayName, activeTab} = this.props;
        const {error, showConfirmModal, subscriptionToDelete, isTemplate} = this.state;

        let confirmDeleteMessage = '';
        confirmDeleteMessage = `Are you sure to delete the subscription ${isTemplate ? 'template' : ''}?`;
        if (subscriptionToDelete && subscriptionToDelete.name) {
            confirmDeleteMessage = `Are you sure to delete the subscription  ${isTemplate ? 'template' : ''} "${subscriptionToDelete.name}"?`;
        }

        let confirmModal = null;
        if (showConfirmModal) {
            confirmModal = (
                <ConfirmModal
                    cancelButtonText={'Cancel'}
                    confirmButtonText={'Delete'}
                    destructive={true}
                    hideCancel={false}
                    message={confirmDeleteMessage}
                    onCancel={this.handleCancelDelete}
                    onConfirm={this.handleConfirmDelete}
                    show={true}
                    title={isTemplate ? 'Subscription Template' : 'Subscription'}
                />
            );
        }

        const showInstanceColumn = this.props.installedInstances.length > 1;
        const columns = (
            <thead>
                <tr>
                    <th scope='col'>{'Name'}</th>
                    <th scope='col'>{'Project'}</th>
                    {showInstanceColumn && <th scope='col'>{'Instance'}</th>}
                    <th scope='col'>{'Actions'}</th>
                </tr>
            </thead>
        );

        const renderTable = (subs: ChannelSubscription[], forTemplates: boolean) => (
            <table className='jira-subscriptions__table'>
                {columns}
                <tbody>
                    {subs.map((sub) => this.renderRow(sub, forTemplates))}
                </tbody>
            </table>
        );

        const showTemplates = activeTab === 'templates';
        const subs = showTemplates ? subscriptionTemplates : channelSubscriptions;
        const panel = subs.length ? renderTable(subs, showTemplates) : (
            <p className='jira-subscriptions__empty'>
                {showTemplates ? 'No templates yet' : 'No subscriptions yet'}
            </p>
        );

        const createButton = (
            <Button
                emphasis='secondary'
                size='small'
                leadingIcon={<Icon glyph={<PlusIcon/>}/>}
                onClick={showTemplates ? this.props.showCreateSubscriptionTemplate : this.props.showCreateChannelSubscription}
            >
                {showTemplates ? 'Create Template' : 'Create Subscription'}
            </Button>
        );

        const tabs = (
            <Tabs
                appearance='underlined'
                activeKey={activeTab}
                onChange={(key) => this.props.onTabChange(key as SubscriptionsTab)}
                controls={createButton}
                tabs={[
                    {key: 'subscriptions', label: 'Subscriptions', id: 'jira-subscriptions-tab', panelId: TAB_PANEL_ID},
                    {key: 'templates', label: 'Templates', id: 'jira-templates-tab', panelId: TAB_PANEL_ID},
                ]}
            />
        );

        return (
            <Modal
                open={this.props.open}
                size='medium'
                title='Jira Subscriptions'
                subtitle={!omitDisplayName && channel && `Channel: ${channel.display_name}`}
                subtitlePlacement='beside'
                headerSlot={tabs}
                onClose={this.props.close}
            >
                {error && (
                    <SectionNotice
                        className='jira-form-notice'
                        role='alert'
                        type='danger'
                        title={error}
                    />
                )}
                <div
                    role='tabpanel'
                    id={TAB_PANEL_ID}
                    aria-labelledby={showTemplates ? 'jira-templates-tab' : 'jira-subscriptions-tab'}
                >
                    {panel}
                </div>
                {confirmModal}
            </Modal>
        );
    }
}
