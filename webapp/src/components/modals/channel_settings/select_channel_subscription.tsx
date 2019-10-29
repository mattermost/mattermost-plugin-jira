import React from 'react';

import {ChannelSubscription} from 'types/model';

import ConfirmModal from 'components/confirm_modal';

import {SharedProps} from './shared_props';

type Props = SharedProps & {
    showEditChannelSubscription: (subscription: ChannelSubscription) => void;
    showCreateChannelSubscription: () => void;
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

    handleCancelDelete = () => {
        this.setState({showConfirmModal: false});
    }

    handleConfirmDelete = () => {
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

    render(): React.ReactElement {
        const {channel, omitDisplayName} = this.props;
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

        const subscriptionRows = (
            <table className='table'>
                <thead>
                    <tr>
                        <th scope='col'>{'Name'}</th>
                        <th scope='col'>{'Actions'}</th>
                    </tr>
                </thead>
                <tbody>
                    {this.props.channelSubscriptions.map((sub, i) => (
                        <tr key={i}>
                            <td
                                key={sub.id}
                                className='select-channel-subscriptions-row'
                            >
                                <span>{sub.name || '(no name)'}</span>
                            </td>
                            <td>
                                <button
                                    className='style--none color--link'
                                    onClick={(): void => this.props.showEditChannelSubscription(sub)}
                                >
                                    {'Edit'}
                                </button>
                                {' - '}
                                <button
                                    className='style--none color--link'
                                    onClick={(): void => this.handleDeleteChannelSubscription(sub)}
                                >
                                    {'Delete'}
                                </button>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        );

        return (
            <div>
                <div className='d-flex justify-content-between align-items-center margin-bottom x3'>
                    {titleMessage}
                    <button
                        className='btn btn-primary'
                        onClick={this.props.showCreateChannelSubscription}
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
