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
}

export default class SelectChannelSubscriptionInternal extends React.PureComponent<Props, State> {
    state = {
        error: null,
        showConfirmModal: false,
    };

    handleDeactivateCancel = () => {
        this.setState({showConfirmModal: false});
    }

    handleConfirmDelete = (sub: ChannelSubscription) => {
        this.setState({showConfirmModal: false});
        this.deleteChannelSubscription(sub);
    }

    handleDeleteChannelSubscription = (): void => {
        this.setState({showConfirmModal: true});
    };

    deleteChannelSubscription = (sub: ChannelSubscription): void => {
        this.props.deleteChannelSubscription(sub).then((res: {error?: {message: string}}) => {
            if (res.error) {
                this.setState({error: res.error.message});
            }
        });
    };

    render(): React.ReactElement {
        const {channel} = this.props;
        const {error, showConfirmModal} = this.state;

        const headerText = `Jira Subscriptions in "${channel.name}"`;

        let errorDisplay = null;
        if (error) {
            errorDisplay = (
                <span className='error'>{error}</span>
            );
        }

        const subscriptionRows = this.props.channelSubscriptions.map((sub) => (
            <div
                key={sub.id}
                className='select-channel-subscriptions-row'
            >
                <ConfirmModal
                    cancelButtonText={'Cancel'}
                    confirmButtonText={'Delete'}
                    confirmButtonClass={'btn btn-danger'}
                    hideCancel={false}
                    message={'Delete Subscription "' + sub.id + '"?'}
                    onCancel={this.handleDeactivateCancel}
                    onConfirm={(): void => this.handleConfirmDelete(sub)}
                    show={showConfirmModal}
                    title={'Subscription'}
                />
                <div className='channel-subscription-id-container'>
                    <span>{sub.id}</span>
                </div>
                <button
                    className='btn btn-info'
                    onClick={(): void => this.props.showEditChannelSubscription(sub)}
                >
                    {'Edit'}
                </button>
                <button
                    className='btn btn-danger'
                    onClick={this.handleDeleteChannelSubscription}
                >
                    {'Delete'}
                </button>
            </div>
        ));

        return (
            <div>
                <h1>{headerText}</h1>
                <button
                    className='btn btn-info'
                    onClick={this.props.showCreateChannelSubscription}
                >
                    {'Create Subscription'}
                </button>
                {errorDisplay}
                {subscriptionRows}
            </div>
        );
    }
}
