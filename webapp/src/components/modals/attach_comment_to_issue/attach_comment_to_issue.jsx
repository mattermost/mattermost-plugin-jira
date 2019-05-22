// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import FormButton from 'components/form_button';
import Loading from 'components/loading';
import ReactSelectSetting from 'components/react_select_setting';
import Input from 'components/input';

import {getProjectValues} from 'jira_issue_metadata';

const initialState = {
    submitting: false,
    projectKey: null,
    issueKey: null,
    error: null,
};

export default class AttachIssueModal extends PureComponent {
    static propTypes = {
        close: PropTypes.func.isRequired,
        create: PropTypes.func.isRequired,
        post: PropTypes.object,
        theme: PropTypes.object.isRequired,
        visible: PropTypes.bool.isRequired,
        jiraIssueMetadata: PropTypes.object,
        fetchJiraIssueMetadata: PropTypes.func.isRequired,
    };

    constructor(props) {
        super(props);
        this.state = initialState;
    }

    componentDidUpdate(prevProps) {
        if (this.props.post && (!prevProps.post || this.props.post.id !== prevProps.post.id)) {
            this.props.fetchJiraIssueMetadata();
        }
    }

    handleCreate = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        const issue = {
            post_id: this.props.post.id,
            issueKey: this.state.issueKey,
        };

        this.setState({submitting: true});

        this.props.create(issue).then((created) => {
            if (created.error) {
                this.setState({error: created.error.message, submitting: false});
                return;
            }
            this.handleClose(e);
        });
    };

    handleClose = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }
        const {close} = this.props;
        this.setState(initialState, close);
    };

    handleProjectChange = (id, value) => {
        const projectKey = value;
        this.setState({
            projectKey,
        });
    }

    handleIssueKeyChange = (id, value) => {
        const issueKey = value;
        this.setState({
            issueKey,
        });
    }

    render() {
        const {post, visible, theme, jiraIssueMetadata} = this.props;
        const {error, submitting} = this.state;
        const style = getStyle(theme);

        if (!visible) {
            return null;
        }

        let component;
        if (error) {
            console.error('render error', error); //eslint-disable-line no-console
        }

        if (!post || !jiraIssueMetadata || !jiraIssueMetadata.projects) {
            component = <Loading/>;
        } else {
            const projectOptions = getProjectValues(jiraIssueMetadata);
            component = (
                <div style={style.modal}>
                    <ReactSelectSetting
                        name={'project'}
                        label={'Project'}
                        required={true}
                        onChange={this.handleProjectChange}
                        placeholder={'Select project'}
                        options={projectOptions}
                        isMuli={false}
                        key={'LT'}
                        value={projectOptions.filter((option) => option.value === this.state.projectKey)}
                    />
                    <Input
                        key='key'
                        id='issueKey'
                        placeholder={'Enter issue key to attach message to, e.g. EXT-20'}
                        label='Issue Key'
                        type='input'
                        onChange={this.handleIssueKeyChange}
                        required={true}
                        disabled={false}
                    />
                    <Input
                        label='Message Attached to Jira Issue'
                        type='textarea'
                        isDisabled={true}
                        value={this.props.post.message}
                        disabled={false}
                    />
                    <br/>
                </div>
            );
        }

        return (
            <Modal
                dialogClassName='modal--scroll'
                show={visible}
                onHide={this.handleClose}
                onExited={this.handleClose}
                bsSize='large'
            >
                <Modal.Header closeButton={true}>
                    <Modal.Title>
                        {'Attach Message to Jira Issue'}
                    </Modal.Title>
                </Modal.Header>
                <form
                    role='form'
                    onSubmit={this.handleCreate}
                >
                    <Modal.Body ref='modalBody'>
                        {component}
                    </Modal.Body>
                    <Modal.Footer>
                        <FormButton
                            type='button'
                            btnClass='btn-default'
                            defaultMessage='Cancel'
                            onClick={this.handleClose}
                        />
                        <FormButton
                            type='submit'
                            btnClass='btn btn-primary'
                            saving={submitting}
                            defaultMessage='Attach'
                            savingMessage='Attaching'
                        >
                            {'Attach'}
                        </FormButton>
                    </Modal.Footer>
                </form>
            </Modal>
        );
    }
}

const getStyle = (theme) => ({
    modal: {
        padding: '1em',
        color: theme.centerChannelColor,
        backgroundColor: theme.centerChannelBg,
    },
    descriptionArea: {
        height: 'auto',
        width: '100%',
        color: '#000',
    },
});
