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
        this.props.deleteChannelSubscription(sub).then((res: { error?: { message: string } }) => {
            if (res.error) {
                this.setState({error: res.error.message});
            }
        });
    };

    render(): React.ReactElement {
        const {channel, omitDisplayName} = this.props;
        const {error, showConfirmModal} = this.state;

        let errorDisplay = null;
        if (error) {
            errorDisplay = (
                <span className='error'>{error}</span>
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
                                    onClick={this.handleDeleteChannelSubscription}
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
                {errorDisplay}
                {subscriptionRows}
            </div>
        );
    }
}
