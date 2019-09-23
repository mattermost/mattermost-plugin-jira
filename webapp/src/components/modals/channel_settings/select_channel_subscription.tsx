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
        this.props.deleteChannelSubscription(sub).then((res: {error?: {message: string}}) => {
            if (res.error) {
                this.setState({error: res.error.message});
            }
        });
    };

    render(): React.ReactElement {
        const {channel} = this.props;
        const {error} = this.state;

        const headerText = `Jira Subscriptions in "${channel.name}"`;

        let errorDisplay = null;
        if (error) {
            errorDisplay = (
                <span className='error'>{error}</span>
            );
        }

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
                {this.props.channelSubscriptions.map((sub) => (
                    <div
                        key={sub.id}
                        className='select-channel-subscriptions-row'
                    >
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
                            onClick={(): void => this.deleteChannelSubscription(sub)}
                        >
                            {'Delete'}
                        </button>
                    </div>
                ))}
            </div>
        );
    }
}
