// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';
import debounce from 'lodash/debounce';

import FormButton from 'components/form_button';
import Loading from 'components/loading';
import ReactSelectSetting from 'components/react_select_setting';
import Input from 'components/input';

import {getProjectValues} from 'jira_issue_metadata';

const initialState = {
    submitting: false,
    projectKey: null,
    issueKey: null,
    textSearchTerms: '',
    error: null,
};

const searchDefaults = 'ORDER BY updated DESC';
const searchDebounceDelay = 400;

export default class AttachIssueModal extends PureComponent {
    static propTypes = {
        close: PropTypes.func.isRequired,
        create: PropTypes.func.isRequired,
        post: PropTypes.object,
        theme: PropTypes.object.isRequired,
        visible: PropTypes.bool.isRequired,
        jiraIssueMetadata: PropTypes.object,
        jiraIssueOptions: PropTypes.array,
        fetchJiraIssueMetadata: PropTypes.func.isRequired,
        fetchJiraIssues: PropTypes.func.isRequired,
    };

    constructor(props) {
        super(props);
        this.state = initialState;
    }

    componentDidUpdate(prevProps) {
        if (this.props.post && (!prevProps.post || this.props.post.id !== prevProps.post.id)) {
            this.props.fetchJiraIssueMetadata();
            this.props.fetchJiraIssues(searchDefaults);
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
        this.setState({
            projectKey: value,
            jiraIssueOptions: [],
        });

        this.searchTermsChanged(value, this.state.textSearchTerms);
    };

    handleIssueKeyChange = ({value}) => {
        this.setState({
            issueKey: value,
        });
    };

    handleIssueSearchTermChange = (id, value) => {
        this.setState({
            textSearchTerms: value,
        });

        this.searchTermsChanged(this.state.projectKey, value);
    };

    searchTermsChanged = debounce((projectKey, text) => {
        const projectSearchTerm = projectKey ? 'project=' + projectKey : '';
        const textEncoded = encodeURIComponent(text.replace(/"/g, '\\"'));
        const textSearchTerm = (textEncoded.length > 0) ? 'text ~ "' + textEncoded + '"' : '';
        const combinedTerms = (projectSearchTerm.length > 0 && textSearchTerm.length > 0) ? projectSearchTerm + ' AND ' + textSearchTerm : projectSearchTerm + textSearchTerm;
        this.props.fetchJiraIssues(combinedTerms + ' ' + searchDefaults);
    }, searchDebounceDelay);

    render() {
        const {post, visible, theme, jiraIssueMetadata, jiraIssueOptions} = this.props;
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
                        isMulti={false}
                        key={'LT'}
                        value={projectOptions.filter((option) => option.value === this.state.projectKey)}
                    />
                    <ReactSelectSetting
                        key={'issueToAttach'}
                        name={'issue'}
                        placeholder={'Select an issue to attach the message to...'}
                        label={'Issue Key'}
                        onChange={this.handleIssueKeyChange}
                        required={true}
                        disabled={false}
                        isMulti={false}
                        isClearable={true}
                        options={jiraIssueOptions}
                        helpText={'Showing the ' + jiraIssueOptions.length + ' most recently changed items.'}
                    />
                    <Input
                        id='issueSearchTerms'
                        placeholder={'Find issues containing the text...'}
                        label='Search for the issues containing:'
                        type='input'
                        onChange={this.handleIssueSearchTermChange}
                        required={false}
                        disabled={false}
                        helpText={'Tips: use AND, OR, *, ?, ~, etc., just like any JQL query.'}
                    />
                    <Input
                        label='Message Attached to Jira Issue'
                        type='textarea'
                        isDisabled={true}
                        value={this.props.post.message}
                        disabled={false}
                        readOnly={true}
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
        height: '460px',
    },
    descriptionArea: {
        height: 'auto',
        width: '100%',
        color: '#000',
    },
});
