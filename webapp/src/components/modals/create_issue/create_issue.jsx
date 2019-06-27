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

import {getProjectValues, getIssueTypes, getIssueValues, getFields} from 'utils/jira_issue_metadata';

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
        fetchJiraIssueMetadata: PropTypes.func.isRequired,
    };

    constructor(props) {
        super(props);

        this.state = initialState;

        this.projectRef = React.createRef();
        this.issueRef = React.createRef();

        this.validator = new Validator();
    }

    componentDidMount() {
        this.validator.addComponent('project', this.projectRef);
        this.validator.addComponent('issue', this.issueRef);
    }

    componentWillUnmount() {
        this.validator.removeComponent('project');
        this.validator.removeComponent('issue');
    }

    componentDidUpdate(prevProps) {
        if (this.props.post && (!prevProps.post || this.props.post.id !== prevProps.post.id)) {
            this.props.fetchJiraIssueMetadata();
            const fields = {...this.state.fields};
            fields.description = this.props.post.message;
            this.setState({fields}); //eslint-disable-line react/no-did-update-set-state
        } else if (this.props.channelId && (this.props.channelId !== prevProps.channelId || this.props.description !== prevProps.description)) {
            this.props.fetchJiraIssueMetadata();
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

    getFieldsNotCovered() {
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
        const fields = {...this.state.fields};
        const issueTypes = getIssueTypes(this.props.jiraIssueMetadata, value);
        const issueType = issueTypes.length && issueTypes[0].id;
        const projectKey = value;
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
        const {post, visible, theme, jiraIssueMetadata} = this.props;
        const {error, submitting} = this.state;
        const style = getStyle(theme);

        if (!visible) {
            return null;
        }

        let issueError = null;
        if (error) {
            issueError = (
                <React.Fragment>
                    <p className='alert alert-danger'>
                        <i
                            className='fa fa-warning'
                            title='Warning Icon'
                        />
                        <span> {error}</span>
                    </p>
                </React.Fragment>
            );
        }
        let component;
        let footer = (
            <React.Fragment>
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
                >
                    {'Create'}
                </FormButton>
            </React.Fragment>
        );

        if (jiraIssueMetadata && jiraIssueMetadata.error) {
            component = (
                <div style={style.modal}>
                    {jiraIssueMetadata.error}
                </div>
            );

            footer = (
                <React.Fragment>
                    <FormButton
                        type='submit'
                        btnClass='btn btn-primary'
                        defaultMessage='Close'
                        onClick={this.handleClose}
                    />
                </React.Fragment>
            );
        } else if (!post || !jiraIssueMetadata || !jiraIssueMetadata.projects) {
            component = <Loading/>;
        } else {
            const issueOptions = getIssueValues(jiraIssueMetadata, this.state.projectKey);
            const projectOptions = getProjectValues(jiraIssueMetadata);
            component = (
                <div style={style.modal}>
                    {issueError}
                    <ReactSelectSetting
                        ref={this.projectRef}
                        name={'project'}
                        label={'Project'}
                        required={true}
                        onChange={this.handleProjectChange}
                        options={projectOptions}
                        isMuli={false}
                        key={'LT'}
                        theme={theme}
                        value={projectOptions.find((option) => option.value === this.state.projectKey)}
                    />
                    <ReactSelectSetting
                        ref={this.issueRef}
                        name={'issue_type'}
                        label={'Issue Type'}
                        required={true}
                        onChange={this.handleIssueTypeChange}
                        options={issueOptions}
                        isMuli={false}
                        theme={theme}
                        value={issueOptions.find((option) => option.value === this.state.issueType)}
                    />
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
                backdrop='static'
                keyboard={false}
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
                    <Modal.Body ref='modalBody'>
                        {component}
                    </Modal.Body>
                    <Modal.Footer>
                        {footer}
                    </Modal.Footer>
                </form>
            </Modal>
        );
    }
}

const getStyle = (theme) => ({
    modal: {
        padding: '1em 1em 0',
        color: theme.centerChannelColor,
        backgroundColor: theme.centerChannelBg,
    },
    descriptionArea: {
        height: 'auto',
        width: '100%',
        color: '#000',
    },
});
