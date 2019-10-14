import React from 'react';

import {ChannelSubscription} from 'types/model';

import {SharedProps} from './shared_props';

type Props = SharedProps & {
    showEditChannelSubscription: (subscription: ChannelSubscription) => void;
    showCreateChannelSubscription: () => void;
};

type State = {
    error: string | null;
}

export default class SelectChannelSubscriptionInternal extends React.PureComponent<Props, State> {
    state = {
        error: null,
    };

    deleteChannelSubscription = (sub: ChannelSubscription): void => {
        this.props.deleteChannelSubscription(sub).then((res: { error?: { message: string } }) => {
            if (res.error) {
                this.setState({ error: res.error.message });
            }
        });
    };

    render(): React.ReactElement {
        const { channel } = this.props;
        const { error } = this.state;

        let errorDisplay = null;
        if (error) {
            errorDisplay = (
                <span className='error'>{error}</span>
            );
        }

        return (
            <div>
                <div className='d-flex justify-content-between align-items-center margin-bottom x3'>
                    <h2 className='text-center'>{'Jira Subscriptions in'} <strong>{channel.name}</strong></h2>
                    <button
                        className='btn btn-primary'
                        onClick={this.props.showCreateChannelSubscription}
                    >
                        {'Create Subscription'}
                    </button>
                </div>
                {errorDisplay}
                {this.props.channelSubscriptions.map((sub) => (
                    <table className='table'>
                        <thead>
                            <tr>
                                <th scope='col'>ID</th>
                                <th scope='col'>Actions</th>
                            </tr>
                        </thead>
                        <tbody>
                            <td
                                key={sub.id}
                                className='select-channel-subscriptions-row'
                            >
                                <span>{sub.id}</span>
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
                                    onClick={(): void => this.deleteChannelSubscription(sub)}
                                >
                                    {'Delete'}
                                </button>
                            </td>
                        </tbody>
                    </table>
                ))}
            </div>
        );
    }
}
