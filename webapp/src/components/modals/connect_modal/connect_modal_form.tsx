// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Modal} from 'react-bootstrap';

import {id as PluginId} from 'manifest';

import FormButton from 'components/form_button';
import JiraInstanceSelector from 'components/jira_instance_selector';

import {getModalStyles} from 'utils/styles';

import {Props} from './props';
export type State = {
    submitting: boolean;
    error: string;
    selectedInstance: string;
};

export default class ConnectModalForm extends PureComponent<Props, State> {
    state = {
        submitting: false,
        error: '',
        selectedInstance: '',
    };

    submit = async (e) => {
        if (e.preventDefault) {
            e.preventDefault();
        }

        const selectedInstance = this.state.selectedInstance;
        if (!selectedInstance) {
            this.setState({error: 'Please select a Jira instance'});
            return;
        }

        if (this.isAlreadyConnectedToInstance(this.state.selectedInstance)) {
            this.setState({error: 'You are already connected to this Jira instance.'});
            return;
        }

        this.props.closeModal();
        this.props.redirectConnect(this.state.selectedInstance);
    }

    isAlreadyConnectedToInstance = (instanceID: string): boolean => {
        return Boolean(this.props.connectedInstances.find((instance) => instance.instance_id === instanceID));
    };

    closeModal = (e) => {
        this.props.closeModal();
    }

    handleInstanceChoice = (instanceID: string) => {
        let error = '';
        if (instanceID && this.isAlreadyConnectedToInstance(instanceID)) {
            error = 'You are already connected to this Jira instance.';
        }

        this.setState({selectedInstance: instanceID, error});
    }

    render(): JSX.Element {
        const style = getModalStyles(this.props.theme);
        const {selectedInstance} = this.state;

        const component = (
            <JiraInstanceSelector
                theme={this.props.theme}
                onChange={this.handleInstanceChoice}
                value={selectedInstance}
            />
        );

        const disableSubmit = !selectedInstance || this.isAlreadyConnectedToInstance(selectedInstance);
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
                    defaultMessage='Connect'
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
                    ref='modalBody'
                >
                    {component}
                </Modal.Body>
                <Modal.Footer style={style.modalFooter}>
                    {footer}
                </Modal.Footer>
            </form>
        );
    }
}
