// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';

import {Button} from '@mattermost/compass-ui/components/button';
import {Modal} from '@mattermost/compass-ui/components/modal';
import {SectionNotice} from '@mattermost/compass-ui/components/section-notice';

import {Post} from '@mattermost/types/posts';
import {Team} from '@mattermost/types/teams';

import {APIResponse, AttachCommentRequest, SavedFieldValues} from 'types/model';

import Input from 'components/input';
import JiraIssueSelector from 'components/jira_issue_selector';
import Validator from 'components/validator';

import JiraInstanceAndProjectSelector from 'components/jira_instance_and_project_selector';

function getPostDisplayMessage(post: Post): string {
    if (post.message) {
        return post.message;
    }

    // If no message, check for file attachments
    const files = post.metadata?.files;
    if (files && files.length > 0) {
        const fileNames = files.map((file) => file.name);
        if (fileNames.length === 1) {
            return `Attached file: ${fileNames[0]}`;
        }
        return `Attached files: ${fileNames.join(', ')}`;
    }

    // Fallback: check file_ids if metadata.files is not populated
    const fileIds = post.file_ids;
    if (fileIds && fileIds.length > 0) {
        if (fileIds.length === 1) {
            return 'Attached file';
        }
        return `Attached ${fileIds.length} files`;
    }

    return '';
}

type Props = {
    close: () => void;
    attachComment: (payload: AttachCommentRequest) => Promise<APIResponse<{}>>;
    post: Post;
    currentTeam: Team;
    visible: boolean;
}

const FORM_ID = 'jira-attach-comment-form';

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

    handleClose = (e?: React.SyntheticEvent) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        this.props.close();
    };

    handleIssueKeyChange = (issueKey: string) => {
        this.setState({issueKey});
    };

    render() {
        const {error, submitting} = this.state;

        const instanceSelector = (
            <JiraInstanceAndProjectSelector
                selectedInstanceID={this.state.instanceID}
                selectedProjectID={''}
                hideProjectSelector={true}
                onInstanceChange={(instanceID: string) => this.setState({instanceID})}
                onProjectChange={(savedValues: SavedFieldValues) => {}}
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
                        value={this.state.issueKey}
                        instanceID={this.state.instanceID}
                    />
                    <Input
                        addValidate={this.validator.addComponent}
                        removeValidate={this.validator.removeComponent}
                        label='Message Attached to Jira Issue'
                        id='jira-attach-comment-message'
                        type='textarea'
                        value={getPostDisplayMessage(this.props.post)}
                        readOnly={true}
                    />
                </div>
            );
        }

        const disableSubmit = !(this.state.instanceID && this.state.issueKey);
        const footer = (
            <>
                <Button
                    emphasis='tertiary'
                    onClick={this.handleClose}
                >
                    {'Cancel'}
                </Button>
                <Button
                    type='submit'
                    form={FORM_ID}
                    emphasis='primary'
                    loading={submitting}
                    disabled={disableSubmit}
                >
                    {'Attach'}
                </Button>
            </>
        );

        return (
            <Modal
                open={this.props.visible}
                size='large'
                title='Attach Message to Jira Issue'
                onClose={this.handleClose}
                footer={footer}
            >
                <form
                    id={FORM_ID}
                    role='form'
                    onSubmit={this.handleSubmit}
                >
                    {error && (
                        <SectionNotice
                            className='jira-form-notice'
                            role='alert'
                            type='danger'
                            title={error}
                        />
                    )}
                    {instanceSelector}
                    {form}
                </form>
            </Modal>
        );
    }
}
