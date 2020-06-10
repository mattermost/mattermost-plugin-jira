// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

import React, {PureComponent} from 'react';
import PropTypes from 'prop-types';
import {Modal} from 'react-bootstrap';

import Validator from 'components/validator';
import JiraFields from 'components/jira_fields';
import FormButton from 'components/form_button';
import Loading from 'components/loading';
import ReactSelectSetting from 'components/react_select_setting';

import {getProjectValues, getIssueValues, getFields} from 'utils/jira_issue_metadata';
import {getModalStyles} from 'utils/styles';

const initialState = {
    submitting: false,
    projectKey: null,
    issueType: null,
    fields: {
        description: '',
        project: {
            key: '',
        },
        issuetype: {
            name: '',
        },
    },
    error: null,
    getMetaDataError: null,
};

export default class CreateIssueModal extends PureComponent {
    static propTypes = {
        close: PropTypes.func.isRequired,
        create: PropTypes.func.isRequired,
        post: PropTypes.object,
        description: PropTypes.string,
        channelId: PropTypes.string,
        currentTeam: PropTypes.object.isRequired,
        theme: PropTypes.object.isRequired,
        visible: PropTypes.bool.isRequired,
        jiraIssueMetadata: PropTypes.object,
        clearIssueMetadata: PropTypes.func.isRequired,
        jiraProjectMetadata: PropTypes.object,
        fetchJiraIssueMetadataForProjects: PropTypes.func.isRequired,
        fetchJiraProjectMetadata: PropTypes.func.isRequired,
    };

    constructor(props) {
        super(props);

        this.state = initialState;

        this.validator = new Validator();
    }

    componentDidUpdate(prevProps) {
        if (this.props.post && (!prevProps.post || this.props.post.id !== prevProps.post.id)) {
            this.props.fetchJiraProjectMetadata().then((fetched) => {
                if (fetched.error) {
                    this.setState({getMetaDataError: fetched.error.message, submitting: false});
                }
            });
            const fields = {...this.state.fields};
            fields.description = this.props.post.message;
            this.setState({fields}); //eslint-disable-line react/no-did-update-set-state
        } else if (this.props.channelId && (this.props.channelId !== prevProps.channelId || this.props.description !== prevProps.description)) {
            this.props.fetchJiraProjectMetadata().then((fetched) => {
                if (fetched.error) {
                    this.setState({getMetaDataError: fetched.error.message, submitting: false});
                }
            });
            const fields = {...this.state.fields};
            fields.description = this.props.description;
            this.setState({fields}); //eslint-disable-line react/no-did-update-set-state
        }
    }

    allowedFields = [
        'project',
        'issuetype',
        'priority',
        'description',
        'summary',
    ];

    allowedSchemaCustom = [
        'com.atlassian.jira.plugin.system.customfieldtypes:textarea',
        'com.atlassian.jira.plugin.system.customfieldtypes:textfield',
        'com.atlassian.jira.plugin.system.customfieldtypes:select',
        'com.atlassian.jira.plugin.system.customfieldtypes:project',

        // 'com.pyxis.greenhopper.jira:gh-epic-link',

        // epic label is 'Epic Name' for cloud instance
        'com.pyxis.greenhopper.jira:gh-epic-label',
    ];

    getFieldsNotCovered = () => {
        const {jiraIssueMetadata} = this.props;
        const myfields = getFields(jiraIssueMetadata, this.state.projectKey, this.state.issueType);

        const fieldsNotCovered = [];

        Object.keys(myfields).forEach((key) => {
            if (myfields[key].required) {
                if ((!myfields[key].schema.custom && !this.allowedFields.includes(key)) ||
                    (myfields[key].schema.custom && !this.allowedSchemaCustom.includes(myfields[key].schema.custom))
                ) {
                    if (!fieldsNotCovered.includes(key)) {
                        // Send down the key and the localized name.
                        fieldsNotCovered.push([key, myfields[key].name]);
                    }
                }
            }
        });
        return fieldsNotCovered;
    }

    handleCreate = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }

        const {post, channelId} = this.props;
        const postId = (post) ? post.id : '';

        if (!this.validator.validate()) {
            return;
        }

        const requiredFieldsNotCovered = this.getFieldsNotCovered();

        const issue = {
            post_id: postId,
            current_team: this.props.currentTeam.name,
            fields: this.state.fields,
            channel_id: channelId,
            required_fields_not_covered: requiredFieldsNotCovered,
        };

        this.setState({submitting: true});

        this.props.create(issue).then((created) => {
            if (created.error) {
                this.setState({error: created.error.message, submitting: false});
                return;
            }
            this.handleClose();
        });
    };

    handleClose = (e) => {
        if (e && e.preventDefault) {
            e.preventDefault();
        }
        const {close} = this.props;
        this.setState(initialState, close);
    };

    handleDescriptionChange = (e) => {
        const description = e.target.value;
        const {fields} = this.state;
        const nFields = {
            ...fields,
            description,
        };

        this.setState({fields: nFields});
    };

    handleProjectChange = (id, value) => {
        const projectKey = value;

        // Clear the current metadata so that we display a loading indicator while we fetch the new metadata
        this.props.clearIssueMetadata();
        this.props.fetchJiraIssueMetadataForProjects([projectKey]).then((fetched) => {
            if (fetched.error) {
                this.setState({getMetaDataError: fetched.error.message, submitting: false});
            }
        });

        const fields = {...this.state.fields};
        const issueTypes = getIssueValues(this.props.jiraProjectMetadata, value);
        const issueType = issueTypes.length && issueTypes[0].value;
        fields.project = {
            key: value,
        };
        fields.issuetype = {
            id: issueType,
        };
        this.setState({
            projectKey,
            issueType,
            fields,
        });
    };

    handleIssueTypeChange = (id, value) => {
        const fields = {...this.state.fields};
        const issueType = value;
        fields.issuetype = {
            id: issueType,
        };
        this.setState({
            issueType,
            fields,
        });
    };

    handleFieldChange = (id, value) => {
        const fields = {...this.state.fields};
        fields[id] = value;
        this.setState({
            fields,
        });
    };

    render() {
        const {visible, theme, jiraIssueMetadata, jiraProjectMetadata} = this.props;
        const {error, getMetaDataError, submitting} = this.state;
        const style = getModalStyles(theme);

        if (!visible) {
            return null;
        }

        const footerClose = (
            <FormButton
                type='submit'
                btnClass='btn btn-primary'
                defaultMessage='Close'
                onClick={this.handleClose}
            />
        );

        let footer = (
            <React.Fragment>
                <FormButton
                    type='button'
                    btnClass='btn-link'
                    defaultMessage='Cancel'
                    onClick={this.handleClose}
                />
                <FormButton
                    id='submit-button'
                    type='submit'
                    btnClass='btn btn-primary'
                    saving={submitting}
                >
                    {'Create'}
                </FormButton>
            </React.Fragment>
        );

        let issueError = null;
        let component;

        // if no getMetaDataError, show fields and allow user to input
        // fields. An error at this point is from a server-side create
        // issue submission and should be displayed (in addition to fields)
        // to the user after clicking the create button.
        if (error) {
            issueError = (
                <p className='alert alert-danger'>
                    <i
                        className='fa fa-warning'
                        title='Warning Icon'
                    />
                    <span> {error}</span>
                </p>
            );
        }

        // if getmetadata fails, only display the error and a close button.
        // user is not going to be able to create a ticket or even a partial
        // ticket. likely a permissions error.
        if (getMetaDataError) {
            component = (
                <p className='alert alert-danger'>
                    <i
                        className='fa fa-warning'
                        title='Warning Icon'
                    />
                    <span> {getMetaDataError}</span>
                </p>
            );
            footer = footerClose;
        } else if ((jiraIssueMetadata && jiraIssueMetadata.error) ||
            (jiraProjectMetadata && jiraProjectMetadata.error)) {
            const msg = (jiraIssueMetadata && jiraIssueMetadata.error) ? jiraIssueMetadata.error : jiraProjectMetadata.error;
            component = (
                <div>
                    {msg}
                </div>
            );
            footer = footerClose;
        } else if (!jiraProjectMetadata || !jiraProjectMetadata.projects) {
            component = <Loading/>;
        } else {
            const issueOptions = getIssueValues(jiraProjectMetadata, this.state.projectKey);
            const projectOptions = getProjectValues(jiraProjectMetadata);

            let fieldsComponent;
            if (!this.state.projectKey) {
                fieldsComponent = null;
            } else if (jiraIssueMetadata) {
                fieldsComponent = (
                    <JiraFields
                        fields={getFields(jiraIssueMetadata, this.state.projectKey, this.state.issueType)}
                        onChange={this.handleFieldChange}
                        values={this.state.fields}
                        allowedFields={this.allowedFields}
                        allowedSchemaCustom={this.allowedSchemaCustom}
                        theme={theme}
                        value={this.state.fields}
                        addValidate={this.validator.addComponent}
                        removeValidate={this.validator.removeComponent}
                    />
                );
            } else {
                fieldsComponent = <Loading/>;
            }

            component = (
                <div>
                    {issueError}
                    <ReactSelectSetting
                        name={'project'}
                        label={'Project'}
                        limitOptions={true}
                        required={true}
                        onChange={this.handleProjectChange}
                        options={projectOptions}
                        isMulti={false}
                        key={'LT'}
                        theme={theme}
                        value={projectOptions.find((option) => option.value === this.state.projectKey)}
                        addValidate={this.validator.addComponent}
                        removeValidate={this.validator.removeComponent}
                    />
                    <ReactSelectSetting
                        name={'issue_type'}
                        label={'Issue Type'}
                        required={true}
                        onChange={this.handleIssueTypeChange}
                        options={issueOptions}
                        isMulti={false}
                        theme={theme}
                        value={issueOptions.find((option) => option.value === this.state.issueType)}
                        addValidate={this.validator.addComponent}
                        removeValidate={this.validator.removeComponent}
                    />
                    {fieldsComponent}
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
                backdrop='static'
            >
                <Modal.Header closeButton={true}>
                    <Modal.Title>
                        {'Create Jira Issue'}
                    </Modal.Title>
                </Modal.Header>
                <form
                    role='form'
                    onSubmit={this.handleCreate}
                >
                    <Modal.Body
                        style={style.modalBody}
                    >
                        {component}
                    </Modal.Body>
                    <Modal.Footer style={style.modalFooter}>
                        {footer}
                    </Modal.Footer>
                </form>
            </Modal>
        );
    }
}
