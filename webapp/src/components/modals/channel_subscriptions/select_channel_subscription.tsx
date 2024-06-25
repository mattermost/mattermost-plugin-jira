import React from 'react';

import {AllProjectMetadata, ChannelSubscription} from 'types/model';

import ConfirmModal from 'components/confirm_modal';

import {SharedProps} from './shared_props';

type Props = SharedProps & {
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
        const projectName = this.getProjectName(sub);

        const showInstanceColumn = this.props.installedInstances.length > 1;

        const alias = this.props.installedInstances.filter((instance) => instance.instance_id === sub.instance_id)[0].alias;
        const instanceName = alias || sub.instance_id;

        if (!forTemplates) {
            return this.renderSubscriptionRow(sub, projectName, showInstanceColumn, instanceName);
        }
        return this.renderSubscriptionTemplateRow(sub, projectName, showInstanceColumn, instanceName);
    };

    renderSubscriptionRow(sub: ChannelSubscription, projectName: string, showInstanceColumn: boolean, instanceName: string): JSX.Element {
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
                        <span>{instanceName}</span>
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

    renderSubscriptionTemplateRow(sub: ChannelSubscription, projectName: string, showInstanceColumn: boolean, instanceName: string): JSX.Element {
        return (
            <tr
                key={sub.id}
                className='select-channel-subscriptions-row'
            >
                <td>{sub.name || '(no name)'}</td>
                <td>{projectName}</td>
                {showInstanceColumn && (
                    <td>{instanceName}</td>
                )}

                <td>
                    <button
                        className='style--none color--link'
                        onClick={() => this.props.showEditSubscriptionTemplate(sub)}
                        type='button'
                    >
                        {'Edit'}
                    </button>
                    {' - '}
                    <button
                        className='style--none color--link'
                        onClick={() => this.handleDeleteChannelSubscription(sub, true)}
                        type='button'
                    >
                        {'Delete'}
                    </button>
                </td>
            </tr>
        );
    }

    render(): React.ReactElement {
        const {channel, channelSubscriptions, subscriptionTemplates, omitDisplayName} = this.props;
        const {error, showConfirmModal, subscriptionToDelete, isTemplate} = this.state;

        let errorDisplay = null;
        if (error) {
            errorDisplay = (
                <span className='error'>{error}</span>
            );
        }

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
                    confirmButtonClass={'btn btn-danger'}
                    hideCancel={false}
                    message={confirmDeleteMessage}
                    onCancel={this.handleCancelDelete}
                    onConfirm={this.handleConfirmDelete}
                    show={true}
                    title={isTemplate ? 'Subscription Template' : 'Subscription'}
                />
            );
        }

        let titleMessage = <h2 className='text-center'>{'Jira Subscriptions in'} <strong>{channel.display_name}</strong></h2>;
        if (omitDisplayName) {
            titleMessage = <h2 className='text-center'>{'Jira Subscriptions'}</h2>;
        }

        const subscriptionTemplateTitle = <h2 className='text-center'>{'Jira Subscription Templates'}</h2>;
        const showInstanceColumn = this.props.installedInstances.length > 1;
        let subscriptionRows;
        let subscriptionTemplateRows;
        const columns = (
            <thead>
                <tr>
                    <th scope='col'>{'Name'}</th>
                    <th
                        className='th-col'
                        scope='col'
                    >{'Project'}</th>
                    {showInstanceColumn &&
                    <th
                        className='th-col'
                        scope='col'
                    >{'Instance'}</th>
                    }
                    <th
                        className='th-col'
                        scope='col'
                    >{'Actions'}</th>
                </tr>
            </thead>
        );
        if (channelSubscriptions.length) {
            subscriptionRows = (
                <table className='table'>
                    {columns}
                    <tbody>
                        {channelSubscriptions.map((element) => (
                            this.renderRow(element, false)
                        ))}
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

        if (subscriptionTemplates.length) {
            subscriptionTemplateRows = (
                <table className='table'>
                    {columns}
                    <tbody>
                        {subscriptionTemplates.map((element) => (
                            this.renderRow(element, true)
                        ))}
                    </tbody>
                </table>
            );
        } else {
            subscriptionTemplateRows = (
                <p>{'Click "Create Template" to create subscription templates.'}</p>
            );
        }

        return (
            <div>
                <div className='d-flex justify-content-between align-items-center margin-bottom x3 title-message'>
                    {titleMessage}
                    <button
                        className='btn btn-primary'
                        onClick={this.props.showCreateChannelSubscription}
                        type='button'
                    >
                        {'Create Subscription'}
                    </button>
                </div>
                {subscriptionRows}
                <div className='d-flex justify-content-between align-items-center margin-bottom x3'>
                    {subscriptionTemplateTitle}
                    <button
                        className='btn btn-primary'
                        onClick={this.props.showCreateSubscriptionTemplate}
                        type='button'
                    >
                        {'Create Template'}
                    </button>
                </div>
                {subscriptionTemplateRows}
                {confirmModal}
                {errorDisplay}
            </div>
        );
    }
}
