# Model Training

## Model - Lifecycle - CREATE/UPDATE/DELETE/GETDETAILS

### CreateModel

Request to create a model, please note as part of MVP1, there will not be any workflows associated with model request
approval

AI consumers , will pass the below details list of addresses that can access this model ( applicable if model is not
public)
is model is public Description of the model

In return the AI consumer will get back a model ID

### UpdateModelAccess

AI consumer can add/remove addresses associated with a given model , can also make this public from private and
viceversa.

### DeleteModel

AI consumer had created and can request that the model be deleted.

### GetModelStatus

Pass the model Id and get back the training status of this model

### GetAllAccessibleModels

An AI consumer can always call back this method to determine the list of models associated to a given address , this can
be used while making inference calls , the AI consumer can pick the model of their choice if multiple models are
available

![](/home/adminaccount/Downloads/trainingflow.png.png)
