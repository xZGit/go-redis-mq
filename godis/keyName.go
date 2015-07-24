package godis





var prefix="godis::"

func Consumers(consumer string) string {   //set
	return  prefix + "comsumers::" +consumer
}


func Producers(producer string) string {    //set
	return  prefix + "producers::" +producer
}


func ProducerService(service string) string {  //set  value:producer
	return  prefix +  "service::" + service
}


func ProducerMsgQueen(producer string) string {   //list
	return  prefix +  "producers::" + producer + "::message"
}




func ConsumerMsgQueen(consumer string) string {  //list
	return  prefix + "comsumers::" + consumer + "::message"
}

