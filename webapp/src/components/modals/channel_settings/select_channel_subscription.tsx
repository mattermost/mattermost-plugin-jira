import React from 'react';

import {ChannelSubscription, AllProjectMetadata} from 'types/model';

import ConfirmModal from 'components/confirm_modal';

import {SharedProps} from './shared_props';

type Props = SharedProps & {
    showEditChannelSubscription: (subscription: ChannelSubscription) => void;
    showCreateChannelSubscription: () => void;
    allProjectMetadata: AllProjectMetadata | null;
};

type State = {
    error: string | null;
    showConfirmModal: boolean;
    subscriptionToDelete: ChannelSubscription | null;
}

export default class SelectChannelSubscriptionInternal extends React.PureComponent<Props, State> {
    state = {
        error: null,
        showConfirmModal: false,
        subscriptionToDelete: null,
    };

    handleCancelDelete = (): void => {
        this.setState({showConfirmModal: false});
    }

    handleConfirmDelete = (): void => {
        this.setState({showConfirmModal: false});
        this.deleteChannelSubscription(this.state.subscriptionToDelete);
    }

    handleDeleteChannelSubscription = (sub: ChannelSubscription): void => {
        this.setState({
            showConfirmModal: true,
            subscriptionToDelete: sub,
        });
    };

    deleteChannelSubscription = (sub: ChannelSubscription): void => {
        this.props.deleteChannelSubscription(sub).then((res: { error?: { message: string } }) => {
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
    }

    renderRow = (sub: ChannelSubscription): JSX.Element => {
        const projectName = this.getProjectName(sub);

        const showInstanceColumn = this.props.installedInstances.length > 1;
        return (
            <tr
                key={sub.id}
                className='select-channel-subscriptions-row'
            >
                <td>
                    <span>{sub.name || '(no name)'}</span>
                </td>
                <td>
                    <span>{projectName}</span>
                </td>
                {showInstanceColumn && (
                    <td>
                        <span>{sub.instance_id}</span>
                    </td>
                )}

                <td>
                    <button
                        className='style--none color--link'
                        onClick={(): void => this.props.showEditChannelSubscription(sub)}
                        type='button'
                    >
                        {'Edit'}
                    </button>
                    {' - '}
                    <button
                        className='style--none color--link'
                        onClick={(): void => this.handleDeleteChannelSubscription(sub)}
                        type='button'
                    >
                        {'Delete'}
                    </button>
                </td>
            </tr>
        );
    }

    render(): React.ReactElement {
        const {channel, channelSubscriptions, omitDisplayName} = this.props;
        const {error, showConfirmModal, subscriptionToDelete} = this.state;

        let errorDisplay = null;
        if (error) {
            errorDisplay = (
                <span className='error'>{error}</span>
            );
        }

        let confirmDeleteMessage = 'Delete Subscription?';
        if (subscriptionToDelete && subscriptionToDelete.name) {
            confirmDeleteMessage = `Delete Subscription "${subscriptionToDelete.name}"?`;
        }

        let confirmModal = null;
        if (showConfirmModal) {
            confirmModal = (
                <ConfirmModal
                    cancelButtonText={'Cancel'}
                    confirmButtonText={'Delete'}
                    confirmButtonClass={'btn btn-danger'}
                    hideCancel={false}
                    message={confirmDeleteMessage}
                    onCancel={this.handleCancelDelete}
                    onConfirm={this.handleConfirmDelete}
                    show={true}
                    title={'Subscription'}
                />
            );
        }

        let titleMessage = <h2 className='text-center'>{'Jira Subscriptions in'} <strong>{channel.display_name}</strong></h2>;
        if (omitDisplayName) {
            titleMessage = <h2 className='text-center'>{'Jira Subscriptions'}</h2>;
        }

        const showInstanceColumn = this.props.installedInstances.length > 1;
        let subscriptionRows;
        if (channelSubscriptions.length) {
            subscriptionRows = (
                <table className='table'>
                    <thead>
                        <tr>
                            <th scope='col'>{'Name'}</th>
                            <th scope='col'>{'Project'}</th>
                            {showInstanceColumn && <th scope='col'>{'Instance'}</th>}
                            <th scope='col'>{'Actions'}</th>
                        </tr>
                    </thead>
                    <tbody>
                        {channelSubscriptions.map(this.renderRow)}
                    </tbody>
                </table>
            );
        } else {
            subscriptionRows = (
                <p>
                    {'Click "Create Subscription" to receive Jira issue notifications in this channel.'}
                </p>
            );
        }

        return (
            <div>
                <div className='d-flex justify-content-between align-items-center margin-bottom x3'>
                    {titleMessage}
                    <button
                        className='btn btn-primary'
                        onClick={this.props.showCreateChannelSubscription}
                        type='button'
                    >
                        {'Create Subscription'}
                    </button>
                </div>
                {confirmModal}
                {errorDisplay}
                {subscriptionRows}
            </div>
        );
    }
}
