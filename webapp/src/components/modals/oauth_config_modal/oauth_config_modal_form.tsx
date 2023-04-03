// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Modal} from 'react-bootstrap';

import {getModalStyles} from 'utils/styles';

import FormButton from 'components/form_button';
import Input from 'components/input';
import Validator from 'components/validator';

import {Props} from './props';
export type State = {
    submitting: boolean;
    error: string;
    instanceUrl: string;
    clientId: string;
    clientSecret: string;
};

export default class OAuthConfigModalForm extends PureComponent<Props, State> {
    private validator = new Validator();
    state = {
        submitting: false,
        error: '',
        instanceUrl: '',
        clientId: '',
        clientSecret: '',
    };

    submit = async (e?: React.FormEvent) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        if (!this.validator.validate()) {
            return;
        }

        if (this.isAlreadyInstalledInstance(this.state.instanceUrl)) {
            this.setState({error: 'You have already installed this Jira instance.'});
            return;
        }

        const config = {
            instance_url: this.state.instanceUrl,
            client_id: this.state.clientId,
            client_secret: this.state.clientSecret,
        };
        this.setState({submitting: true});
        this.props.configure(config).then(({error}) => {
            if (error) {
                this.setState({error: error.message, submitting: false});
                return;
            }
            this.props.closeModal();
        });
    }

    isAlreadyInstalledInstance = (instanceID: string): boolean => {
        return Boolean(this.props.installedInstances.find((instance) => instance.instance_id === instanceID));
    };

    handleInstanceChange = (id: string, instanceID: string) => {
        if (instanceID === this.state.instanceUrl) {
            return;
        }

        let error = '';
        if (instanceID && this.isAlreadyInstalledInstance(instanceID)) {
            error = 'You have already installed this Jira instance.';
        }
        this.setState({instanceUrl: instanceID, error});
    };

    handleClientIdChange = (id: string, value: string) => {
        this.setState({clientId: value});
    };

    handleClientSecretChange = (id: string, value: string) => {
        this.setState({clientSecret: value});
    };

    closeModal = (e?: Event) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        this.props.closeModal();
    }

    render(): JSX.Element {
        const style = getModalStyles(this.props.theme);

        const disableSubmit = false;
        const footer = (
            <React.Fragment>
                <FormButton
                    type='button'
                    btnClass='btn-link'
                    defaultMessage='Cancel'
                    onClick={this.closeModal}
                />
                <FormButton
                    type='submit'
                    btnClass='btn btn-primary'
                    defaultMessage='Configure'
                    disabled={disableSubmit}
                    saving={this.state.submitting}
                />
                <p className={'error-text'}>
                    {this.state.error}
                </p>
            </React.Fragment>
        );

        return (
            <form
                role='form'
                onSubmit={this.submit}
            >
                <Modal.Body
                    style={style.modalBody}
                >
                    <Input
                        addValidate={this.validator.addComponent}
                        removeValidate={this.validator.removeComponent}
                        label='Jira Cloud organization'
                        helpText='Enter a Jira Cloud URL (typically, `https://yourorg.atlassian.net`), or just the organization part, `yourorg`'
                        type='input'
                        placeholder='https://yourorg.atlassian.net'
                        disabled={false}
                        required={true}
                        value={this.state.instanceUrl}
                        onChange={this.handleInstanceChange}
                    />
                    <Input
                        addValidate={this.validator.addComponent}
                        removeValidate={this.validator.removeComponent}
                        label='Jira OAuth Client ID'
                        helpText='The client ID for the OAuth app registered with Jira'
                        type='input'
                        disabled={false}
                        required={true}
                        value={this.state.clientId}
                        onChange={this.handleClientIdChange}
                    />
                    <Input
                        addValidate={this.validator.addComponent}
                        removeValidate={this.validator.removeComponent}
                        label='Jira OAuth Client Secret'
                        helpText='The client secret for the OAuth app registered with Jira'
                        type='input'
                        disabled={false}
                        required={true}
                        value={this.state.clientSecret}
                        onChange={this.handleClientSecretChange}
                    />
                </Modal.Body>
                <Modal.Footer style={style.modalFooter}>
                    {footer}
                </Modal.Footer>
            </form>
        );
    }
}
