// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import {Modal} from 'react-bootstrap';

import {Post} from 'mattermost-redux/types/posts';
import {Team} from 'mattermost-redux/types/teams';
import {Theme} from 'mattermost-redux/types/preferences';

import {APIResponse, AttachCommentRequest} from 'types/model';

import {getModalStyles} from 'utils/styles';

import FormButton from 'components/form_button';
import Input from 'components/input';
import JiraIssueSelector from 'components/jira_issue_selector';
import Validator from 'components/validator';

import JiraInstanceAndProjectSelector from 'components/jira_instance_and_project_selector';

type Props = {
    close: () => void;
    attachComment: (payload: AttachCommentRequest) => Promise<APIResponse<{}>>;
    post: Post;
    currentTeam: Team;
    theme: Theme;
}

type State = {
    submitting: boolean;
    issueKey: string | null;
    textSearchTerms: string;
    error: string | null;
    instanceID: string;
}

export default class AttachCommentToIssueForm extends PureComponent<Props, State> {
    private validator = new Validator();
    state = {
        submitting: false,
        issueKey: null,
        textSearchTerms: '',
        error: null,
        instanceID: '',
    } as State;

    handleSubmit = (e: React.FormEvent) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        if (!this.validator.validate()) {
            return;
        }

        const issue = {
            post_id: this.props.post.id,
            current_team: this.props.currentTeam.name,
            issueKey: this.state.issueKey as string,
            instance_id: this.state.instanceID as string,
        };

        this.setState({submitting: true});
        this.props.attachComment(issue).then(({error}) => {
            if (error) {
                this.setState({error: error.message, submitting: false});
            } else {
                this.handleClose();
            }
        });
    };

    handleClose = (e?: Event) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        this.props.close();
    };

    handleIssueKeyChange = (issueKey: string) => {
        this.setState({issueKey});
    };

    render() {
        const {theme} = this.props;
        const {error, submitting} = this.state;
        const style = getModalStyles(theme);

        const instanceSelector = (
            <JiraInstanceAndProjectSelector
                selectedInstanceID={this.state.instanceID}
                selectedProjectID={''}
                hideProjectSelector={true}
                onInstanceChange={(instanceID: string) => this.setState({instanceID})}
                onProjectChange={(projectKey: string) => {}}
                theme={this.props.theme}
                addValidate={this.validator.addComponent}
                removeValidate={this.validator.removeComponent}
                onError={(err: string) => this.setState({error: err})}
            />
        );

        let form;
        if (this.state.instanceID) {
            form = (
                <div>
                    <JiraIssueSelector
                        addValidate={this.validator.addComponent}
                        removeValidate={this.validator.removeComponent}
                        onChange={this.handleIssueKeyChange}
                        required={true}
                        theme={theme}
                        error={error}
                        value={this.state.issueKey}
                        instanceID={this.state.instanceID}
                    />
                    <Input
                        addValidate={this.validator.addComponent}
                        removeValidate={this.validator.removeComponent}
                        label='Message Attached to Jira Issue'
                        type='textarea'
                        isDisabled={true}
                        value={this.props.post.message}
                        disabled={false}
                        readOnly={true}
                    />
                </div>
            );
        }

        const disableSubmit = !(this.state.instanceID && this.state.issueKey);
        return (
            <form
                role='form'
                onSubmit={this.handleSubmit}
            >
                <Modal.Body
                    style={style.modalBody}
                >
                    {instanceSelector}
                    {form}
                </Modal.Body>
                <Modal.Footer style={style.modalFooter}>
                    <FormButton
                        type='button'
                        btnClass='btn-link'
                        defaultMessage='Cancel'
                        onClick={this.handleClose}
                    />
                    <FormButton
                        type='submit'
                        btnClass='btn btn-primary'
                        saving={submitting}
                        defaultMessage='Attach'
                        savingMessage='Attaching'
                        disabled={disableSubmit}
                    >
                        {'Attach'}
                    </FormButton>
                </Modal.Footer>
            </form>
        );
    }
}
